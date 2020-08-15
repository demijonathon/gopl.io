// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// +build example
//
// This build tag means that "go install golang.org/x/exp/shiny/..." doesn't
// install this example program. Use "go run main.go" to run it or "go install
// -tags=example" to install it.
// Basic is a basic example of a graphical application.
package main

import (
	//	"fmt"
	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/imageutil"
	"golang.org/x/exp/shiny/screen"
	//	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"image"
	"image/color"
	"log"
	"math"
)

var (
	black    = color.RGBA{0x00, 0x00, 0x00, 0xff}
	white    = color.RGBA{0xff, 0xff, 0xff, 0xff}
	blue0    = color.RGBA{0x00, 0x00, 0x1f, 0xff}
	blue1    = color.RGBA{0x00, 0x00, 0x3f, 0xff}
	darkGray = color.RGBA{0x3f, 0x3f, 0x3f, 0xff}
	green    = color.RGBA{0x00, 0x7f, 0x00, 0x7f}
	red      = color.RGBA{0xff, 0x00, 0x00, 0x7f}
	yellow   = color.RGBA{0x3f, 0x3f, 0x00, 0x3f}
	cos30    = math.Cos(math.Pi / 6)
	sin30    = math.Sin(math.Pi / 6)
)

func main() {
	driver.Main(func(s screen.Screen) {
		w, err := s.NewWindow(&screen.NewWindowOptions{
			Title: "Reaction-Diffusion",
		})
		if err != nil {
			log.Fatal(err)
		}
		defer w.Release()
		var sz size.Event

		for {
			e := w.NextEvent()
			// This print message is to help programmers learn what events this
			// example program generates. A real program shouldn't print such
			// messages; they're not important to end users.
			//format := "got %#v\n"
			//if _, ok := e.(fmt.Stringer); ok {
			//	format = "got %v\n"
			//}
			//fmt.Printf(format, e)

			switch e := e.(type) {
			case lifecycle.Event:
				if e.To == lifecycle.StageDead {
					return
				}
			case key.Event:
				if e.Code == key.CodeEscape {
					return
				}
			case paint.Event:
				const inset = 10
				for _, r := range imageutil.Border(sz.Bounds(), inset) {
					w.Fill(r, blue0, screen.Src)
				}
				w.Fill(sz.Bounds().Inset(inset), blue1, screen.Src)
				//w.Upload(image.Point{20, 0}, b, b.Bounds())
				//w.Fill(image.Rect(50, 50, 350, 120), red, screen.Over)
				w.Fill(image.Rect(150, 150, 250, 220), white, screen.Src)

				w.Publish()
			case size.Event:
				sz = e
			case error:
				log.Print(e)
			}
		}
	})
}
