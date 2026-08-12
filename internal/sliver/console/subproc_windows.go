//go:build windows

package console

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"siren/internal/envvars"

	"golang.org/x/sys/windows"
)

// subprocCommandTerminator submits a line to the subprocess console.
// Console input under ConPTY delivers Enter as a carriage return.
const subprocCommandTerminator = "\r"

// StartConsole spawns a subprocess that runs a real sliver client
// console (readline + all commands) attached to a fresh ConPTY pseudo
// console. The subprocess gets the pseudoconsole as its console at
// creation, so sliver's readline (CONIN$/console API) works exactly as
// it does in Windows Terminal. Returns a jobID the frontend uses for I/O.
func (s *Service) StartConsole(sessionID string) (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}

	// The subprocess needs the operator config to reconnect. We pass the
	// serialized ClientConfig via a %TEMP% file, deleted after read.
	cfgPath, err := writeConfigForSubproc(s.rpc.Config)
	if err != nil {
		return "", err
	}

	cpty, err := startConPTY(winQuoteArgs(append([]string{self}, ConsoleModeFlag, cfgPath, sessionID)), 100, 30)
	if err != nil {
		_ = os.Remove(cfgPath)
		return "", err
	}

	id := s.subproc.newJobID()
	job := &subprocJob{id: id, sessionID: sessionID, proc: cpty, pty: cpty}
	s.subproc.add(job)
	s.drainPendingCommands(job)
	go s.pumpSubproc(job)
	go s.watchConsole(job, cfgPath)
	return id, nil
}

func (s *Service) ResizeConsole(jobID string, cols, rows int) error {
	job := s.subproc.get(jobID)
	if job == nil {
		return os.ErrClosed
	}
	if cols <= 0 || rows <= 0 {
		return nil
	}
	cpty, ok := job.pty.(*winConPTY)
	if !ok {
		return os.ErrInvalid
	}
	return cpty.Resize(int16(cols), int16(rows))
}

// winConPTY wraps a Windows pseudo console and the process attached to
// it. It serves as both the consolePTY (Read/Write/Close over the pipe
// ends the parent keeps) and the consoleProc (Wait/Kill/ExitCode on the
// child process handle).
type winConPTY struct {
	hpc    windows.Handle // pseudoconsole
	in     *os.File       // write end: GUI -> console input
	out    *os.File       // read end: console output -> GUI
	ptyIn  windows.Handle // conhost's copy of the input pipe (kept alive)
	ptyOut windows.Handle // conhost's copy of the output pipe (kept alive)

	// mu guards the process handle so a Kill racing Close can't
	// terminate an unrelated process that reused the handle value.
	mu       sync.Mutex
	proc     windows.Handle
	exited   bool
	exitCode int
}

// startConPTY creates a pseudoconsole of cols x rows and launches
// cmdLine (pre-quoted) attached to it.
func startConPTY(cmdLine string, cols, rows int16) (*winConPTY, error) {
	// Pipe ends are created non-inheritable: ConPTY dups what it needs
	// into conhost, and the child receives console handles through the
	// pseudoconsole attribute rather than handle inheritance.
	var inRead, inWrite windows.Handle
	if err := windows.CreatePipe(&inRead, &inWrite, nil, 0); err != nil {
		return nil, fmt.Errorf("conpty input pipe: %w", err)
	}
	var outRead, outWrite windows.Handle
	if err := windows.CreatePipe(&outRead, &outWrite, nil, 0); err != nil {
		windows.CloseHandle(inRead)
		windows.CloseHandle(inWrite)
		return nil, fmt.Errorf("conpty output pipe: %w", err)
	}

	c := &winConPTY{}
	size := windows.Coord{X: cols, Y: rows}
	if err := windows.CreatePseudoConsole(size, inRead, outWrite, 0, &c.hpc); err != nil {
		windows.CloseHandle(inRead)
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		windows.CloseHandle(outWrite)
		return nil, fmt.Errorf("create pseudo console: %w", err)
	}
	// Keep our copies of the PTY-side ends open for the lifetime of the
	// pseudoconsole (they are closed in Close). CreatePseudoConsole dupes
	// them into conhost, but holding them guards against conhost handle
	// lifetime edge cases and matches the battle-tested conpty library.
	c.ptyIn = inRead
	c.ptyOut = outWrite

	if err := c.spawn(cmdLine); err != nil {
		windows.ClosePseudoConsole(c.hpc)
		windows.CloseHandle(inRead)
		windows.CloseHandle(outWrite)
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return nil, err
	}

	c.in = os.NewFile(uintptr(inWrite), "conpty-in")
	c.out = os.NewFile(uintptr(outRead), "conpty-out")
	return c, nil
}

// spawn starts cmdLine attached to the pseudoconsole via the
// PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE process attribute.
func (c *winConPTY) spawn(cmdLine string) error {
	attrList, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return fmt.Errorf("conpty attribute list: %w", err)
	}
	defer attrList.Delete()
	// PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE takes the HPCON by value as
	// lpValue (see the "Creating a Pseudoconsole session" docs, which
	// pass hPC directly rather than &hPC). The double dereference spells
	// "reinterpret handle as pointer" without a uintptr→Pointer
	// conversion, which go vet's unsafeptr check would flag.
	hpc := c.hpc
	if err := attrList.Update(
		windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE,
		*(*unsafe.Pointer)(unsafe.Pointer(&hpc)),
		unsafe.Sizeof(hpc),
	); err != nil {
		return fmt.Errorf("conpty attribute: %w", err)
	}

	siEx := &windows.StartupInfoEx{}
	siEx.Cb = uint32(unsafe.Sizeof(*siEx))
	siEx.ProcThreadAttributeList = attrList.List()
	// STARTF_USESTDHANDLES with zeroed std handles tells CreateProcess to
	// substitute the pseudoconsole's handles for stdin/stdout/stderr.
	// Without this a GUI-subsystem build (wails release builds use
	// -H windowsgui) gets NULL std handles: the child produces no output
	// and its readline spins on a dead input handle.
	siEx.StartupInfo.Flags |= windows.STARTF_USESTDHANDLES

	cmdLinePtr, err := windows.UTF16PtrFromString(cmdLine)
	if err != nil {
		return fmt.Errorf("conpty command line: %w", err)
	}
	var pi windows.ProcessInformation
	passthrough := envvars.BuildPassthroughEnv()
	windowsSystemVars := []string{
		"SystemRoot", "TEMP", "TMP", "COMSPEC", "USERPROFILE",
		"ALLUSERSPROFILE", "PROCESSOR_ARCHITECTURE", "NUMBER_OF_PROCESSORS",
		"OS", "PATHEXT", "WINDIR",
	}
	for _, name := range windowsSystemVars {
		if v, ok := os.LookupEnv(name); ok {
			passthrough = append(passthrough, name+"="+v)
		}
	}
	envBlock := syscall.StringToUTF16Ptr(strings.Join(passthrough, "\x00") + "\x00")
	err = windows.CreateProcess(
		nil, cmdLinePtr, nil, nil, false,
		windows.EXTENDED_STARTUPINFO_PRESENT,
		envBlock, nil, &siEx.StartupInfo, &pi,
	)
	if err != nil {
		return fmt.Errorf("spawn console subprocess: %w", err)
	}
	windows.CloseHandle(pi.Thread)
	c.mu.Lock()
	c.proc = pi.Process
	c.mu.Unlock()
	return nil
}

func (c *winConPTY) Read(p []byte) (int, error)  { return c.out.Read(p) }
func (c *winConPTY) Write(p []byte) (int, error) { return c.in.Write(p) }

func (c *winConPTY) Resize(cols, rows int16) error {
	return windows.ResizePseudoConsole(c.hpc, windows.Coord{X: cols, Y: rows})
}

func (c *winConPTY) Wait() error {
	c.mu.Lock()
	proc := c.proc
	c.mu.Unlock()
	if proc == 0 {
		return fmt.Errorf("console subprocess not started")
	}
	if _, err := windows.WaitForSingleObject(proc, windows.INFINITE); err != nil {
		return err
	}
	var code uint32
	if err := windows.GetExitCodeProcess(proc, &code); err != nil {
		return err
	}
	c.mu.Lock()
	c.exited = true
	c.exitCode = int(code)
	c.mu.Unlock()
	return nil
}

func (c *winConPTY) Kill() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.proc == 0 || c.exited {
		return nil
	}
	return windows.TerminateProcess(c.proc, 1)
}

func (c *winConPTY) ExitCode() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.exited {
		return -1
	}
	return c.exitCode
}

func (c *winConPTY) Close() error {
	// Closing the pseudoconsole disconnects conhost, which breaks both
	// pipes — that unblocks the pump goroutine's Read even if output
	// was still buffered.
	windows.ClosePseudoConsole(c.hpc)
	if c.out != nil {
		_ = c.out.Close()
	}
	if c.in != nil {
		_ = c.in.Close()
	}
	if c.ptyIn != 0 {
		windows.CloseHandle(c.ptyIn)
	}
	if c.ptyOut != 0 {
		windows.CloseHandle(c.ptyOut)
	}
	c.mu.Lock()
	if c.proc != 0 {
		windows.CloseHandle(c.proc)
		c.proc = 0
	}
	c.mu.Unlock()
	return nil
}

// winQuoteArgs joins arguments into a CreateProcess command line using
// CommandLineToArgvW-compatible quoting.
func winQuoteArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = winQuoteArg(arg)
	}
	return strings.Join(quoted, " ")
}

func winQuoteArg(s string) string {
	if s != "" && !strings.ContainsAny(s, " \t\"") {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	slashes := 0
	for _, r := range s {
		switch r {
		case '\\':
			slashes++
			continue
		case '"':
			b.WriteString(strings.Repeat(`\`, slashes*2+1))
			b.WriteByte('"')
			slashes = 0
			continue
		}
		b.WriteString(strings.Repeat(`\`, slashes))
		slashes = 0
		b.WriteRune(r)
	}
	b.WriteString(strings.Repeat(`\`, slashes*2))
	b.WriteByte('"')
	return b.String()
}
