// License: GPLv3 Copyright: 2023, Kovid Goyal, <kovid at kovidgoyal.net>

package benchmark

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"kitty/tools/cli"
	"kitty/tools/tty"
	"kitty/tools/tui/graphics"
	"kitty/tools/tui/loop"
	"kitty/tools/utils"

	"golang.org/x/exp/slices"
)

var _ = fmt.Print

type Options struct {
	Repetitions    int
	WithScrollback bool
}

const reset = "\x1b]\x1b\\\x1bc"
const ascii_printable = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ \n\t `~!@#$%^&*()_+-=[]{}\\|;:'\",<.>/?"
const chinese_lorem_ipsum = `
旦海司有幼雞讀松鼻種比門真目怪少：扒裝虎怕您跑綠蝶黃，位香法士錯乙音造活羽詞坡村目園尺封鳥朋；法松夕點我冬停雪因科對只貓息加黃住蝶，明鴨乾春呢風乙時昔孝助？小紅女父故去。
飯躲裝個哥害共買去隻把氣年，己你校跟飛百拉！快石牙飽知唱想土人吹象毛吉每浪四又連見、欠耍外豆雞秋鼻。住步帶。
打六申幾麼：或皮又荷隻乙犬孝習秋還何氣；幾裏活打能花是入海乙山節會。種第共後陽沒喜姐三拍弟海肖，行知走亮包，他字幾，的木卜流旦乙左杯根毛。
您皮買身苦八手牛目地止哥彩第合麻讀午。原朋河乾種果「才波久住這香松」兄主衣快他玉坐要羽和亭但小山吉也吃耳怕，也爪斗斥可害朋許波怎祖葉卜。
行花兩耍許車丟學「示想百吃門高事」不耳見室九星枝買裝，枝十新央發旁品丁青給，科房火；事出出孝肉古：北裝愛升幸百東鼻到從會故北「可休笑物勿三游細斗」娘蛋占犬。我羊波雨跳風。
牛大燈兆新七馬，叫這牙後戶耳、荷北吃穿停植身玩間告或西丟再呢，他禾七愛干寺服石安：他次唱息它坐屋父見這衣發現來，苗會開條弓世者吃英定豆哭；跳風掃叫美神。
寸再了耍休壯植己，燈錯和，蝶幾欠雞定和愛，司紅後弓第樹會金拉快喝夕見往，半瓜日邊出讀雞苦歌許開；發火院爸乙；四帶亮錯鳥洋個讀。
`
const misc_unicode = `
‘’“”‹›«»‚„ 😀😛😇😈😉😍😎😮👍👎 —–§¶†‡©®™ →⇒•·°±−×÷¼½½¾
…µ¢£€¿¡¨´¸ˆ˜ ÀÁÂÃÄÅÆÇÈÉÊË ÌÍÎÏÐÑÒÓÔÕÖØ ŒŠÙÚÛÜÝŸÞßàá âãäåæçèéêëìí
îïðñòóôõöøœš ùúûüýÿþªºαΩ∞
`

var opts Options

func benchmark_data(data string, opts Options) (duration time.Duration, sent_data_size int, err error) {
	term, err := tty.OpenControllingTerm(tty.SetRaw)
	if err != nil {
		return 0, 0, err
	}
	defer term.RestoreAndClose()
	state := loop.TerminalStateOptions{Alternate_screen: !opts.WithScrollback}
	if _, err = term.WriteString(state.SetStateEscapeCodes()); err != nil {
		return
	}
	defer func() { _, _ = term.WriteString(state.ResetStateEscapeCodes() + reset) }()
	lock := sync.Mutex{}
	const count = 3
	goroutine_started := make(chan byte)

	go func() {
		lock.Lock()
		defer lock.Unlock()
		buf := make([]byte, 8192)
		var data []byte
		q := []byte(strings.Repeat("\x1b[0n", count))
		goroutine_started <- 'y'
		for !bytes.Contains(data, q) {
			n, err := term.Read(buf)
			if err != nil {
				break
			}
			data = append(data, buf[:n]...)
		}
	}()
	<-goroutine_started

	start := time.Now()
	repeat_count := opts.Repetitions
	for ; repeat_count > 0; repeat_count-- {
		if _, err = term.WriteString(data); err != nil {
			return
		}
		sent_data_size += len(data)
	}
	if _, err = term.WriteString(strings.Repeat("\x1b[5n", count)); err != nil {
		return
	}
	lock.Lock()
	duration = time.Since(start) / time.Duration(opts.Repetitions)
	lock.Unlock()
	return
}

var rand_src = sync.OnceValue(func() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
})

func random_string_of_bytes(n int, alphabet string) string {
	b := make([]byte, n)
	al := len(alphabet)
	src := rand_src()
	for i := 0; i < n; i++ {
		b[i] = alphabet[src.Intn(al)]
	}
	return utils.UnsafeBytesToString(b)
}

type result struct {
	desc     string
	data_sz  int
	duration time.Duration
}

func simple_ascii() (r result, err error) {
	data := random_string_of_bytes(1024*2048+13, ascii_printable)
	duration, data_sz, err := benchmark_data(data, opts)
	if err != nil {
		return result{}, err
	}
	return result{"Only ASCII chars", data_sz, duration}, nil
}

func unicode() (r result, err error) {
	data := strings.Repeat(chinese_lorem_ipsum+misc_unicode, 1024)
	duration, data_sz, err := benchmark_data(data, opts)
	if err != nil {
		return result{}, err
	}
	return result{"Unicode chars", data_sz, duration}, nil
}

func ascii_with_csi() (r result, err error) {
	const sz = 1024*1024 + 17
	out := make([]byte, 0, sz+48)
	src := rand_src()
	chunk := ""
	for len(out) < sz {
		q := src.Intn(100)
		switch {
		case (q < 10):
			chunk = random_string_of_bytes(src.Intn(72)+1, ascii_printable)
		case (10 <= q && q < 30):
			chunk = "\x1b[m;\x1b[?1h\x1b[H"
		case (30 <= q && q < 40):
			chunk = "\x1b[1;2;3;4:3;31m"
		case (40 <= q && q < 50):
			chunk = "\x1b[38:5:24;48:2:125:136:147m"
		case (50 <= q && q < 60):
			chunk = "\x1b[58;5;44;2m"
		case (60 <= q && q < 80):
			chunk = "\x1b[m;\x1b[10A\x1b[3E\x1b[2K"
		case (80 <= q && q < 100):
			chunk = "\x1b[39m;\x1b[10`a\x1b[100b\x1b[?1l"
		}
		out = append(out, utils.UnsafeStringToBytes(chunk)...)
	}
	duration, data_sz, err := benchmark_data(utils.UnsafeBytesToString(out), opts)
	if err != nil {
		return result{}, err
	}
	return result{"CSI codes with few chars", data_sz, duration}, nil
}

func images() (r result, err error) {
	g := graphics.GraphicsCommand{}
	g.SetImageId(12345)
	g.SetQuiet(graphics.GRT_quiet_silent)
	g.SetAction(graphics.GRT_action_transmit)
	g.SetFormat(graphics.GRT_format_rgba)
	const dim = 1024
	g.SetDataWidth(dim)
	g.SetDataHeight(dim)
	g.DisableCompression = true // dont want to measure the speed of zlib
	b := strings.Builder{}
	b.Grow(8 * dim * dim)
	_ = g.WriteWithPayloadTo(&b, make([]byte, 4*dim*dim))
	g.SetAction(graphics.GRT_action_delete)
	g.SetDelete(graphics.GRT_free_by_id)
	_ = g.WriteWithPayloadTo(&b, nil)
	data := b.String()
	duration, data_sz, err := benchmark_data(data, opts)
	if err != nil {
		return result{}, err
	}
	return result{"Images", data_sz, duration}, nil
}

func long_escape_codes() (r result, err error) {
	data := random_string_of_bytes(8024, ascii_printable)
	// OSC 6 is document reporting which kitty ignores after parsing
	data = strings.Repeat("\x1b]6;"+data+"\x07", 1024)
	duration, data_sz, err := benchmark_data(data, opts)
	if err != nil {
		return result{}, err
	}
	return result{"Long escape codes", data_sz, duration}, nil
}

var divs = []time.Duration{
	time.Duration(1), time.Duration(10), time.Duration(100), time.Duration(1000)}

func round(d time.Duration, digits int) time.Duration {
	switch {
	case d > time.Second:
		d = d.Round(time.Second / divs[digits])
	case d > time.Millisecond:
		d = d.Round(time.Millisecond / divs[digits])
	case d > time.Microsecond:
		d = d.Round(time.Microsecond / divs[digits])
	}
	return d
}

func present_result(r result, col_width int) {
	rate := float64(r.data_sz) / r.duration.Seconds()
	rate /= 1024. * 1024.
	f := fmt.Sprintf("%%-%ds", col_width)
	fmt.Printf("  "+f+" : %-10v @ \x1b[32m%-7.1f\x1b[m MB/s\n", r.desc, round(r.duration, 2), rate)
}

func all_benchamrks() []string {
	return []string{
		"ascii", "unicode", "csi", "images", "long_escape_codes",
	}
}

func main(args []string) (err error) {
	if len(args) == 0 {
		args = all_benchamrks()
	}
	var results []result
	var r result
	// First warm up the terminal by getting it to render all chars so that font rendering
	// time is not polluting the benchmarks.
	w := Options{Repetitions: 1}
	if _, _, err = benchmark_data(ascii_printable+chinese_lorem_ipsum+misc_unicode, w); err != nil {
		return err
	}
	time.Sleep(time.Second / 2)

	if slices.Index(args, "ascii") >= 0 {
		if r, err = simple_ascii(); err != nil {
			return err
		}
		results = append(results, r)
	}

	if slices.Index(args, "unicode") >= 0 {
		if r, err = unicode(); err != nil {
			return err
		}
		results = append(results, r)
	}

	if slices.Index(args, "csi") >= 0 {
		if r, err = ascii_with_csi(); err != nil {
			return err
		}
		results = append(results, r)
	}

	if slices.Index(args, "long_escape_codes") >= 0 {
		if r, err = long_escape_codes(); err != nil {
			return err
		}
		results = append(results, r)
	}

	if slices.Index(args, "images") >= 0 {
		if r, err = images(); err != nil {
			return err
		}
		results = append(results, r)
	}

	fmt.Print(reset)
	fmt.Println(
		"These results measure the time it takes the terminal to fully parse all the data sent to it. Note that not all data transmitted will be displayed as input parsing is typically asynchronous with rendering in high performance terminals.")
	fmt.Println()
	fmt.Println("Results:")
	mlen := 10
	for _, r := range results {
		mlen = max(mlen, len(r.desc))
	}
	for _, r := range results {
		present_result(r, mlen)
	}
	return
}

func EntryPoint(root *cli.Command) {
	sc := root.AddSubCommand(&cli.Command{
		Name:             "__benchmark__",
		ShortDescription: "Run various benchmarks",
		HelpText:         "To run only particular benchmarks, specify them on the command line from the set: " + strings.Join(all_benchamrks(), ", "),
		Usage:            "[options] [optional benchmark to run ...]",
		Hidden:           true,
		Run: func(cmd *cli.Command, args []string) (ret int, err error) {
			if err = cmd.GetOptionValues(&opts); err != nil {
				return 1, err
			}
			opts.Repetitions = max(1, opts.Repetitions)
			if err = main(args); err != nil {
				ret = 1
			}
			return
		},
	})
	sc.Add(cli.OptionSpec{
		Name:    "--repetitions",
		Default: "10",
		Type:    "int",
		Help:    "The number of repetitions of each benchmark",
	})
	sc.Add(cli.OptionSpec{
		Name: "--with-scrollback",
		Type: "bool-set",
		Help: "Use the main screen instead of the alt screen so speed of scrollback is also tested",
	})
}
