package handler

import (
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
)

func TerminalWebSocket(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()

		websocket.Message.Send(ws, printPrompt())

		for {
			// Read message (command) from client
			msg := ""
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				c.Logger().Error("WebSocket read error:", err)
				break
			}

			// Process the command
			output, err := executeCommand(msg)
			if err != nil {
				c.Logger().Error("Command execution error:", err)
				websocket.Message.Send(ws, "Error executing command")
				continue
			}

			// Send output back to client
			websocket.Message.Send(ws, output)
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func executeCommand(commandStr string) (string, error) {
	cmdArgs := strings.Fields(commandStr)
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	output, err := cmd.CombinedOutput()
	return string(output) + "\n" + printPrompt(), err
}

func printPrompt() string {
	hostname, _ := os.Hostname()
	currentUser, _ := user.Current()
	pwd, _ := os.Getwd()
	prompt := currentUser.Username + "@" + hostname + ":" + pwd + "$ "

	return prompt
}
