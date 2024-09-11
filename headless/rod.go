package headless

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"time"
)

type RodOptions struct {
	Headless bool   `json:"headless"`
	Proxy    string `json:"proxy"`
	Debug    bool   `json:"debug"`
}

var MyDevice = devices.Device{
	Title:          "Chrome computer",
	Capabilities:   []string{"touch", "mobile"},
	UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
	AcceptLanguage: "en",
	Screen: devices.Screen{
		DevicePixelRatio: 2,
		Horizontal: devices.ScreenSize{
			Width:  1500,
			Height: 900,
		},
		Vertical: devices.ScreenSize{
			Width:  1500,
			Height: 900,
		},
	},
}

func NewRod(options *RodOptions) (l *launcher.Launcher, browser *rod.Browser) {
	l = launcher.New().
		Headless(options.Headless).
		Set("ignore-certificate-errors").
		Delete("disable-component-extensions-with-background-pages").
		Set("disable-extensions").
		Append("disable-features", "BlinkGenPropertyTrees").
		Set("hide-scrollbars").
		Set("mute-audio").
		Set("no-default-browser-check").
		Delete("no-startup-window").
		Set("password-store", "basic").
		Set("safebrowsing-disable-auto-update").
		Set("disable-gpu").
		Set("no-default-browser-check").
		Set("disable-images", "true").
		Set("enable-automation", "false").                     // 防止监测 webdriver
		Set("disable-blink-features", "AutomationControlled"). // 禁用 blink 特征，绕过了加速乐检测
		NoSandbox(true)
	if options.Proxy != "" {
		l.Set(flags.ProxyServer, options.Proxy)
	}
	//args := l.FormatArgs()
	//fmt.Println(args)
	browser = rod.New().ControlURL(l.MustLaunch()).MustConnect()
	if options.Debug {
		browser.Trace(true).SlowMotion(2 * time.Second)
	}
	browser.DefaultDevice(MyDevice)
	return
}
