// Package preference 用户偏好记忆。
//
// 以键值对形式存储用户偏好（姓名 / 喜好 等结构化属性）。
// 写入有两条通道：LLM NER 异步提取（准）+ 规则兜底同步提取（即时一致）。
//
// 规则兜底只认「我」做主语的第一人称陈述句（我喜欢 X / 我爱 X / 我叫 X / 我的名字是 X），
// 否定句、第三人称、疑问句都会被过滤；详见 extractPreference 与 validPrefValue。
//
// 并发安全：所有公共方法持锁。直接读 .Data 字段（旧代码 / JSON 序列化）请用 Snapshot()。
package preference

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// ── 规则抽取的常量与正则 ────────────────────────────────────────────────
//
// 关键设计：触发词必须紧跟在「我」之后（中间只允许副词），因此
// 「我不喜欢 X」「他很喜欢 X」这类**否定**与**第三人称**天然不匹配——
// 不需要额外的排除逻辑，也就不会把它们错记成用户偏好。
const (
	maxPreferenceLen = 20 // 喜好类 value 的最大 rune 数
	maxNameLen       = 12 // 姓名类 value 的最大 rune 数
)

// prefValueStops 是 value 的终止符：命中即截断。
// 避免把「我喜欢猫，也喜欢狗」整句当成一个偏好值。
const prefValueStops = "，,。.；;：:！!？?、\n\r\t（）()【】[]《》\"'“”‘’"

// prefRules 按顺序尝试；命中即返回。顺序敏感——「名字是/名字叫」必须排在「我叫」之前。
var prefRules = []struct {
	re     *regexp.Regexp
	key    string
	maxLen int
}{
	{regexp.MustCompile(`我(?:的)?名字(?:是|叫)(.+)`), "姓名", maxNameLen},
	{regexp.MustCompile(`我(?:很|也|还|就|真的|特别|非常|超|挺|一直)?(?:喜欢|爱)(.+)`), "喜好", maxPreferenceLen},
	{regexp.MustCompile(`我叫(.+)`), "姓名", maxNameLen},
}

// questionPrefixes 疑问词开头说明这是提问而非陈述，不能当作偏好。
var questionPrefixes = []string{"什么", "啥", "哪", "谁", "怎么", "怎样", "多少", "几", "吗"}

// nameStopPrefixes 用于排除「我叫我朋友来」这类使动句（此处的「叫」是"召唤"不是"命名"）。
var nameStopPrefixes = []string{"我", "你", "他", "她", "它", "大家", "朋友"}

// trailingParticles 句尾语气词，清洗时去掉。
const trailingParticles = "吧呢啊呀哦嘛啦噢"

// Preference 以键值对形式存储用户偏好信息
type Preference struct {
	mu   sync.RWMutex
	Data map[string]string `json:"data"`
}

// New 创建用户偏好存储
func New() *Preference {
	return &Preference{Data: make(map[string]string)}
}

// Save 保存单条偏好
func (p *Preference) Save(key, value string) {
	if key == "" || value == "" {
		return
	}
	p.mu.Lock()
	p.Data[key] = value
	p.mu.Unlock()
}

// SaveBatch 批量保存偏好（从 LLM 提取结果）
func (p *Preference) SaveBatch(kvs map[string]string) {
	p.mu.Lock()
	for k, v := range kvs {
		if k != "" && v != "" {
			p.Data[k] = v
		}
	}
	p.mu.Unlock()
}

// Get 安全读取单条偏好
func (p *Preference) Get(key string) (string, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	v, ok := p.Data[key]
	return v, ok
}

// Snapshot 返回偏好数据的浅拷贝（持读锁）
func (p *Preference) Snapshot() map[string]string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	cp := make(map[string]string, len(p.Data))
	for k, v := range p.Data {
		cp[k] = v
	}
	return cp
}

// ExtractAndSave 从对话文本中用规则提取偏好（兜底，LLM 提取优先）。
// 抽取成功会写入内存并返回归一化后的 key/value，供调用方回显。
//
// 纯函数部分见 extractPreference，便于单测；本方法只负责「抽取 + 落库」。
func (p *Preference) ExtractAndSave(msg string) (key, value string, ok bool) {
	key, value, ok = extractPreference(msg)
	if !ok {
		return "", "", false
	}
	p.Save(key, value)
	return key, value, true
}

// extractPreference 无副作用地从文本中抽取偏好，返回归一化后的 key/value。
// 步骤：正则匹配 → value 清洗（截断/去语气词）→ 合法性校验。
func extractPreference(msg string) (key, value string, ok bool) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return "", "", false
	}

	for _, rule := range prefRules {
		m := rule.re.FindStringSubmatch(msg)
		if len(m) < 2 {
			continue
		}
		v := cleanPrefValue(m[1])
		if !validPrefValue(rule.key, v, rule.maxLen) {
			continue // 命中但值非法（疑问句/超长/使动句），继续试下一条规则
		}
		return rule.key, v, true
	}
	return "", "", false
}

// cleanPrefValue 清洗 value：截断到第一个终止符，去掉首尾空白与句尾语气词。
// 长度校验交由 validPrefValue 统一负责。
func cleanPrefValue(raw string) string {
	if i := strings.IndexAny(raw, prefValueStops); i >= 0 {
		raw = raw[:i] // i 一定落在 rune 边界上，按字节切安全
	}
	raw = strings.TrimSpace(raw)
	raw = strings.TrimRight(raw, trailingParticles)
	return strings.TrimSpace(raw)
}

// validPrefValue 校验清洗后的 value 是否可以作为一条偏好落库。
func validPrefValue(key, v string, maxLen int) bool {
	if v == "" {
		return false
	}
	n := len([]rune(v))
	if n == 0 || n > maxLen {
		return false
	}
	// 疑问句不是陈述：「我喜欢什么音乐？」「我叫什么名字？」都不该被记住
	for _, q := range questionPrefixes {
		if strings.HasPrefix(v, q) {
			return false
		}
	}
	if strings.ContainsAny(v, "？?") {
		return false
	}
	// 使动句排除：仅作用于「我叫 X」这条规则
	if key == "姓名" {
		for _, s := range nameStopPrefixes {
			if strings.HasPrefix(v, s) {
				return false
			}
		}
	}
	return true
}

// BuildContext 将偏好数据格式化为给 LLM 的上下文字符串。
// 按 key 排序输出，保证每轮 prompt 前缀稳定（map 遍历顺序随机，会导致 prompt 抖动）。
func (p *Preference) BuildContext() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.Data) == 0 {
		return ""
	}
	keys := make([]string, 0, len(p.Data))
	for k := range p.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	items := make([]string, 0, len(keys))
	for _, k := range keys {
		items = append(items, fmt.Sprintf("%s: %s", k, p.Data[k]))
	}
	return "【用户偏好】\n" + strings.Join(items, "\n")
}
