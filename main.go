package main

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	MIN_VALID_PORT     = 1
	MAX_VALID_PORT     = 65535
	REFRESH_DELAY      = 100 * time.Millisecond
	STATUS_DURATION    = 2 * time.Second
	SUCCESS_DURATION   = 2 * time.Second
	ERROR_DURATION     = 3 * time.Second
	BRIEF_STATUS       = 1 * time.Second
)

type ProcessInfo struct {
	portNumber    string
	procName      string
	processID     string
	addr          string
}

type NetworkService struct {
	processes     []ProcessInfo
	activeIndex   int
	duplicateMap  map[string]bool
}

type TerminalInterface struct {
	app           *tview.Application
	processView   *tview.List
	messageArea   *tview.TextView
	mainContainer *tview.Flex
	service       *NetworkService
	messageTimer  *time.Timer
}

func newNetworkService() *NetworkService {
	return &NetworkService{
		processes:    make([]ProcessInfo, 0),
		activeIndex:  0,
		duplicateMap: make(map[string]bool),
	}
}

func (ns *NetworkService) scanPorts() error {
	ns.processes = make([]ProcessInfo, 0)
	ns.duplicateMap = make(map[string]bool)

	command := exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-n", "-P")
	output, err := command.Output()
	if err != nil {
		log.Printf("Failed to scan ports: %v", err)
		return fmt.Errorf("network scanning operation failed")
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "LISTEN") {
			process := ns.parseLine(line)
			if ns.isProcessValid(process) {
				ns.processes = append(ns.processes, process)
			}
		}
	}

	if scanner.Err() != nil {
		log.Printf("Scan error: %v", scanner.Err())
		return fmt.Errorf("port data processing failed")
	}

	return nil
}

func (ns *NetworkService) parseLine(line string) ProcessInfo {
	fields := strings.Fields(line)
	process := ProcessInfo{}

	if len(fields) >= 9 {
		process.procName = fields[0]
		process.processID = fields[1]
		process.addr = fields[8]

		// simple split instead of regex for port
		parts := strings.Split(process.addr, ":")
		if len(parts) > 1 {
			process.portNumber = parts[len(parts)-1]
		}
	}

	return process
}

func (ns *NetworkService) isProcessValid(process ProcessInfo) bool {
	if process.portNumber == "" {
		return false
	}

	if ns.duplicateMap[process.portNumber] {
		return false
	}

	portValue, parseErr := strconv.Atoi(process.portNumber)
	if parseErr != nil {
		log.Printf("Bad port: %s", process.portNumber)
		return false
	}

	if portValue < MIN_VALID_PORT {
		return false
	}

	if portValue > MAX_VALID_PORT {
		return false
	}

	ns.duplicateMap[process.portNumber] = true
	return true
}

func (ns *NetworkService) processCount() int {
	return len(ns.processes)
}

func (ns *NetworkService) processAtIndex(index int) *ProcessInfo {
	if index < 0 {
		return nil
	}

	if index >= len(ns.processes) {
		return nil
	}

	return &ns.processes[index]
}

func newTerminalInterface() *TerminalInterface {
	app := tview.NewApplication()
	listView := tview.NewList().ShowSecondaryText(false)
	statusView := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)

	headerView := tview.NewTextView().
		SetText("Network Port Manager").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	container := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(headerView, 2, 1, false).
		AddItem(listView, 0, 1, true).
		AddItem(statusView, 3, 1, false)

	app.SetRoot(container, true).SetFocus(listView)

	service := newNetworkService()

	return &TerminalInterface{
		app:           app,
		processView:   listView,
		messageArea:   statusView,
		mainContainer: container,
		service:       service,
	}
}

func (ti *TerminalInterface) showMessage(text string, duration time.Duration) {
	ti.messageArea.SetText(text)

	if ti.messageTimer != nil {
		ti.messageTimer.Stop()
	}

	ti.messageTimer = time.AfterFunc(duration, func() {
		ti.app.QueueUpdateDraw(func() {
			ti.showDefaultStatus()
		})
	})
}

func (ti *TerminalInterface) showDefaultStatus() {
	if ti.service.processCount() == 0 {
		ti.messageArea.SetText("No listening ports detected.\nCommands: r=Refresh, q=Exit")
		return
	}

	ti.messageArea.SetText("Navigation: ↑/↓ Move | k=Kill Process | r=Refresh | q=Exit")
}

func (ti *TerminalInterface) refreshProcessList() {
	scanErr := ti.service.scanPorts()
	if scanErr != nil {
		log.Printf("Refresh failed: %v", scanErr)
		ti.showMessage("Port scanning failed. Please try again.", ERROR_DURATION)
		return
	}

	ti.processView.Clear()

	for _, process := range ti.service.processes {
		displayText := fmt.Sprintf("[red]Port: %s[white] | Process: %s | PID: %s",
			process.portNumber, process.procName, process.processID)
		ti.processView.AddItem(displayText, "", 0, nil)
	}

	if ti.service.processCount() > 0 {
		ti.processView.SetCurrentItem(0)
	}

	if ti.messageTimer == nil {
		ti.showDefaultStatus()
	}
}

func (ti *TerminalInterface) selectedProcess() *ProcessInfo {
	if ti.service.processCount() == 0 {
		return nil
	}

	currentIndex := ti.processView.GetCurrentItem()
	return ti.service.processAtIndex(currentIndex)
}

func (ti *TerminalInterface) configureKeyBindings() {
	ti.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			if event.Rune() == 'q' {
				ti.app.Stop()
				return nil
			}

			if event.Rune() == 'Q' {
				ti.app.Stop()
				return nil
			}

			if event.Rune() == 'r' {
				ti.refresh()
				return nil
			}

			if event.Rune() == 'R' {
				ti.refresh()
				return nil
			}

			if event.Rune() == 'k' {
				ti.killSelectedProcess()
				return nil
			}

			if event.Rune() == 'K' {
				ti.killSelectedProcess()
				return nil
			}
		}

		if event.Key() == tcell.KeyUp {
			return event
		}

		if event.Key() == tcell.KeyDown {
			return event
		}

		if event.Key() == tcell.KeyCtrlC {
			ti.app.Stop()
			return nil
		}

		return event
	})
}

func (ti *TerminalInterface) refresh() {
	ti.showMessage("Scanning network ports...", BRIEF_STATUS)

	go func() {
		time.Sleep(REFRESH_DELAY)
		ti.app.QueueUpdateDraw(func() {
			ti.refreshProcessList()
		})
	}()
}

func (ti *TerminalInterface) killSelectedProcess() {
	selectedProcess := ti.selectedProcess()

	if selectedProcess == nil {
		ti.showMessage("No process selected for termination!", STATUS_DURATION)
		return
	}

	ti.showMessage(fmt.Sprintf("Terminating process on port %s...", selectedProcess.portNumber), STATUS_DURATION)

	go func() {
		killErr := terminateProcess(selectedProcess.processID)
		ti.app.QueueUpdateDraw(func() {
			if killErr != nil {
				ti.showMessage("Process termination failed. Check permissions.", ERROR_DURATION)
				log.Printf("Kill failed on port %s (PID %s): %v",
					selectedProcess.portNumber, selectedProcess.processID, killErr)
				return
			}

			ti.showMessage(fmt.Sprintf("Port %s process terminated successfully!", selectedProcess.portNumber), SUCCESS_DURATION)
			log.Printf("Killed process on port %s (PID %s)",
				selectedProcess.portNumber, selectedProcess.processID)
			ti.refreshProcessList()
		})
	}()
}

func (ti *TerminalInterface) run() error {
	ti.refreshProcessList()
	ti.configureKeyBindings()
	return ti.app.Run()
}

func validatePID(pid string) error {
	if pid == "" {
		log.Printf("Empty PID")
		return fmt.Errorf("process identifier cannot be empty")
	}

	pidValue, parseErr := strconv.Atoi(pid)
	if parseErr != nil {
		log.Printf("Bad PID: %s", pid)
		return fmt.Errorf("invalid process identifier format")
	}

	if pidValue <= 0 {
		log.Printf("Invalid PID: %d", pidValue)
		return fmt.Errorf("process identifier must be positive")
	}

	return nil
}

func sendSignal(pid string, signal string) error {
	command := exec.Command("kill", signal, pid)
	execErr := command.Run()

	if execErr != nil {
		log.Printf("Signal %s failed for %s: %v", signal, pid, execErr)
		return fmt.Errorf("signal execution failed")
	}

	return nil
}

func terminateProcess(pid string) error {
	validationErr := validatePID(pid)
	if validationErr != nil {
		return validationErr
	}

	termErr := sendSignal(pid, "-TERM")
	if termErr != nil {
		log.Printf("TERM failed for %s, trying KILL", pid)

		killErr := sendSignal(pid, "-KILL")
		if killErr != nil {
			log.Printf("KILL failed for %s: %v", pid, killErr)
			return fmt.Errorf("unable to terminate process")
		}
	}

	log.Printf("Terminated PID %s", pid)
	return nil
}

// TODO: Add unit tests for port validation
func main() {
	terminalApp := newTerminalInterface()

	runErr := terminalApp.run()
	if runErr != nil {
		log.Printf("App start failed: %v", runErr)
		fmt.Println("Unable to initialize application. Verify system compatibility.")
	}
}
