package preference

import "testing"

// TestExtractPreference 覆盖规则兜底抽取的「该命中」与「必须不命中」两类用例。
//
// 重点回归：旧实现用 strings.Contains(msg, "我喜欢") 判断，会把
// 「我不喜欢周杰伦」也记成 喜好=周杰伦（"我不喜欢" 里含子串 "我喜欢"），
// 并且用 SplitN 取「喜欢」之后的整串，会把整句话当成偏好值。
func TestExtractPreference(t *testing.T) {
	cases := []struct {
		name      string
		msg       string
		wantKey   string
		wantValue string
		wantOK    bool
	}{
		// ── 应该命中 ──
		{name: "基本喜好", msg: "我喜欢周杰伦的音乐", wantKey: "喜好", wantValue: "周杰伦的音乐", wantOK: true},
		{name: "带副词", msg: "我很喜欢周杰伦", wantKey: "喜好", wantValue: "周杰伦", wantOK: true},
		{name: "也字副词", msg: "我也喜欢喝咖啡", wantKey: "喜好", wantValue: "喝咖啡", wantOK: true},
		{name: "我爱句式", msg: "我爱中国", wantKey: "喜好", wantValue: "中国", wantOK: true},
		{name: "逗号截断", msg: "我喜欢猫，也喜欢狗", wantKey: "喜好", wantValue: "猫", wantOK: true},
		{name: "句号截断", msg: "我喜欢周杰伦的音乐。", wantKey: "喜好", wantValue: "周杰伦的音乐", wantOK: true},
		{name: "句尾语气词", msg: "我喜欢跑步啊", wantKey: "喜好", wantValue: "跑步", wantOK: true},
		{name: "我叫姓名", msg: "我叫张三", wantKey: "姓名", wantValue: "张三", wantOK: true},
		{name: "我叫带从句", msg: "我叫张三，今年二十岁", wantKey: "姓名", wantValue: "张三", wantOK: true},
		{name: "我的名字是", msg: "我的名字是李四", wantKey: "姓名", wantValue: "李四", wantOK: true},
		{name: "我的名字叫", msg: "我的名字叫王五", wantKey: "姓名", wantValue: "王五", wantOK: true},

		// ── 否定句必须不命中（旧实现的 bug）──
		{name: "不喜欢", msg: "我不喜欢周杰伦", wantOK: false},
		{name: "并不喜欢", msg: "我并不喜欢周杰伦", wantOK: false},
		{name: "从来不喜欢", msg: "我从来不喜欢周杰伦", wantOK: false},
		{name: "我不爱", msg: "我不爱吃香菜", wantOK: false},

		// ── 第三人称必须不命中 ──
		{name: "他很喜欢", msg: "他很喜欢周杰伦", wantOK: false},
		{name: "她喜欢", msg: "她喜欢周杰伦", wantOK: false},
		{name: "朋友喜欢", msg: "我朋友喜欢周杰伦", wantOK: false},

		// ── 疑问句必须不命中 ──
		{name: "问喜好", msg: "我喜欢什么音乐？", wantOK: false},
		{name: "问名字", msg: "我叫什么名字？", wantOK: false},

		// ── 其他不该命中的情况 ──
		{name: "使动句", msg: "我叫我朋友来吃饭", wantOK: false},
		{name: "无关文本", msg: "今天天气不错", wantOK: false},
		{name: "空串", msg: "", wantOK: false},
		{name: "只有空白", msg: "   ", wantOK: false},
		{name: "超长值", msg: "我喜欢这个特别特别特别特别特别特别特别特别特别长的东西", wantOK: false},
		{name: "姓名超长", msg: "我叫一二三四五六七八九十十一十二十三", wantOK: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key, value, ok := extractPreference(tc.msg)
			if ok != tc.wantOK {
				t.Fatalf("extractPreference(%q) ok = %v, want %v (got %q=%q)",
					tc.msg, ok, tc.wantOK, key, value)
			}
			if !tc.wantOK {
				return
			}
			if key != tc.wantKey || value != tc.wantValue {
				t.Fatalf("extractPreference(%q) = (%q, %q), want (%q, %q)",
					tc.msg, key, value, tc.wantKey, tc.wantValue)
			}
		})
	}
}

// TestExtractAndSave_WritesOnlyWhenHit 验证落库与返回值的联动：
// 命中写入且可读回；未命中不得写入任何键。
func TestExtractAndSave_WritesOnlyWhenHit(t *testing.T) {
	p := New()

	key, value, ok := p.ExtractAndSave("我喜欢周杰伦的音乐")
	if !ok || key != "喜好" || value != "周杰伦的音乐" {
		t.Fatalf("命中用例失败: key=%q value=%q ok=%v", key, value, ok)
	}
	if got, found := p.Get("喜好"); !found || got != "周杰伦的音乐" {
		t.Fatalf("命中后未写入内存: got=%q found=%v", got, found)
	}

	// 否定句：不应返回 ok，也不应覆盖已有偏好
	if _, _, ok := p.ExtractAndSave("我不喜欢周杰伦"); ok {
		t.Fatal("否定句不应被抽取为偏好")
	}
	if got, _ := p.Get("喜好"); got != "周杰伦的音乐" {
		t.Fatalf("否定句污染了已存偏好: 喜好=%q", got)
	}

	// 无关文本不产生任何键
	p.ExtractAndSave("帮我看看这个报错")
	p.ExtractAndSave("")
	if _, found := p.Get("姓名"); found {
		t.Fatal("无关文本不应写入姓名")
	}
	if n := len(p.Snapshot()); n != 1 {
		t.Fatalf("内存中应只有 1 条偏好，实际 %d 条", n)
	}
}

// TestBuildContext_StableOrder 验证多键输出按 key 排序，避免 prompt 前缀抖动。
func TestBuildContext_StableOrder(t *testing.T) {
	p := New()
	p.Save("城市", "东莞")
	p.Save("姓名", "梁展诚")
	p.Save("喜好", "周杰伦")

	want := "【用户偏好】\n喜好: 周杰伦\n城市: 东莞\n姓名: 梁展诚"
	for i := 0; i < 50; i++ {
		if got := p.BuildContext(); got != want {
			t.Fatalf("第 %d 次输出不稳定:\n got=%q\nwant=%q", i, got, want)
		}
	}
}
