package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
)

type TrainerOutput struct {
	gm     *gocui.Gui
	onQuit func()
}

var (
	LOG_VIEW_NAME    = "log"
	STATUS_VIEW_NAME = "status"

	BOTTOM_PANEL_SIZE = 3

	LOG_LINE_BUFFER = make([]string, 0)
	MAX_LINES       = 60
)

func NewTrainerOutput() *TrainerOutput {
	return &TrainerOutput{}
}

func (ui *TrainerOutput) Run() error {
	gm, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		panic(err)
	}

	ui.gm = gm

	gm.SetManagerFunc(layout)

	if err := gm.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		ui.Close()
		panic(err)
	}

	// Main loop is blocking
	if err := gm.MainLoop(); err != nil && err != gocui.ErrQuit {
		ui.Close()
		panic(err)
	}

	ui.Close()

	return nil
}

func (ui *TrainerOutput) Close() error {
	ui.gm.Close()

	ui.onQuit()

	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if _, err := g.SetView(LOG_VIEW_NAME, -1, -1, maxX, maxY-BOTTOM_PANEL_SIZE); err == nil {
		return err
	}

	if _, err := g.SetView(STATUS_VIEW_NAME, -1, maxY-BOTTOM_PANEL_SIZE, maxX, maxY); err == nil {
		return err
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func (ui *TrainerOutput) LogAgent(msg string) error {
	return ui.LogInfo(msg)
}

func (ui *TrainerOutput) LogDebug(msg string) error {
	// Nothing
	return nil
}

func (ui *TrainerOutput) OnQuit(fn func()) {
	ui.onQuit = fn
}

func (ui *TrainerOutput) LogInfo(msg string) error {
	view, err := ui.gm.View(LOG_VIEW_NAME)

	if err != nil {
		return err
	}

	ui.gm.Update(func(g *gocui.Gui) error {
		view.Clear()

		// Prepend with current date
		time := time.Now().Format(time.StampMilli)

		if len(LOG_LINE_BUFFER) > MAX_LINES {
			LOG_LINE_BUFFER = LOG_LINE_BUFFER[:MAX_LINES]
		}

		LOG_LINE_BUFFER = append([]string{time + " - " + msg}, LOG_LINE_BUFFER...)

		out := strings.Join(LOG_LINE_BUFFER, "\n")

		fmt.Fprintln(view, out)
		return nil
	})

	return nil
}

func (ui *TrainerOutput) LogGameStatus(msg string) error {
	view, err := ui.gm.View(STATUS_VIEW_NAME)

	if err != nil {
		return err
	}

	ui.gm.Update(func(g *gocui.Gui) error {
		view.Clear()

		fmt.Fprintln(view, msg)
		return nil
	})

	return nil
}
