// Copyright 2020 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"regexp"
	"strconv"
	"strings"
	"supertuxkart/api"
	"supertuxkart/gsemanager"
	"supertuxkart/logger"
	"syscall"
	"time"

	sdk "agones.dev/agones/sdks/go"
	"github.com/hpcloud/tail"
	"math/rand"
)

// logLocation is the path to the location of the SuperTuxKart log file
const logLocation = "/.config/supertuxkart/config-0.10/server_config.log"

func signalHandler() {
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	localPid := os.Getpid()
	sig := <-sigChan

	log.Println("caught sig, exit", localPid, sig)
	time.Sleep(3 * time.Second)
	os.Exit(0)
}

func startGrpcServer() int {
	// 启动grpc server，监听agent回调
	grpcServer := api.GetRpcService()
	grpcServer.StartGrpcServer()
	grpcPort := grpcServer.GetGrpcPort()

	// 返回 grpc port
	return grpcPort
}

func startHttpServer() int {
	// 启动 http 服务，方便调试
	httpProcess := api.NewHttpProcess()
	httpProcess.StartHttpServer()
	clientPort := httpProcess.GetHttpPort()

	return clientPort
}

// main intercepts the log file of the SuperTuxKart gameserver and uses it
// to determine if the game server is ready or not.
func main() {
	// 启动Grpc Server
	grpcPort := startGrpcServer()

	// 随机端口
	rand.Seed(2)
	clientPort := 20000 + rand.Intn(10000)

	log.SetPrefix("[wrapper] ")
	input := flag.String("i", "", "the command and arguments to execute the server binary")

	// Since player tracking is not on by default, it is behind this flag.
	// If it is off, still log messages about players, but don't actually call the player tracking functions.
	enablePlayerTracking := flag.Bool("player-tracking", false, "If true, player tracking will be enabled.")
	flag.Parse()

	log.Println("Starting wrapper for SuperTuxKart")

	cmdString := strings.Split(*input, " ")
	cmdString = append(cmdString, "--port="+strconv.Itoa(clientPort))
	command, args := cmdString[0], cmdString[1:]

	log.Printf("Command being run for SuperTuxKart server: %s \n", cmdString)

	cmd := exec.Command(command, args...) // #nosec
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		log.Fatalf("error starting cmd: %v", err)
	}

	log.Println("Connecting to Gse with the SDK")
	gseManager := gsemanager.GetGseManager()
	err := gseManager.ProcessReady([]string{"/local/game/log/log.txt"}, int32(clientPort), int32(grpcPort))
	if err != nil {
		logger.Fatal("processready fail")
	}

	log.Println("Starting health checking")
	//go doHealth(s)

	// SuperTuxKart refuses to output to foreground, so we're going to
	// poll the server log.
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("could not get home dir: %v", err)
	}

	t := &tail.Tail{}
	// Loop to make sure the log has been created. Sometimes it takes a few seconds
	for i := 0; i < 10; i++ {
		time.Sleep(time.Second)

		t, err = tail.TailFile(path.Join(home, logLocation), tail.Config{Follow: true})
		if err != nil {
			log.Print(err)
			continue
		} else {
			break
		}
	}
	defer t.Cleanup()
	for line := range t.Lines {
		// Don't use the logger here. This would add multiple prefixes to the logs. We just want
		// to show the supertuxkart logs as they are, and layer the wrapper logs in with them.
		fmt.Println(line.Text)
		action, player := handleLogLine(line.Text)
		switch action {
		case "READY":
			log.Print("log to mark server ready")
		case "PLAYERJOIN":
			if player == nil {
				log.Print("could not determine player")
				break
			}
			if *enablePlayerTracking {
				log.Print("enablePlayerTracking")
			}
		case "PLAYERLEAVE":
			if player == nil {
				log.Print("could not determine player")
				break
			}
		case "SHUTDOWN":
			os.Exit(0)
		}
	}
	log.Fatal("tail ended")
}

// doHealth sends the regular Health Pings
func doHealth(sdk *sdk.SDK) {
	tick := time.Tick(2 * time.Second)
	for {
		if err := sdk.Health(); err != nil {
			log.Fatalf("could not send health ping: %v", err)
		}
		<-tick
	}
}

// handleLogLine compares the log line to a series of regexes to determine if any action should be taken.
// TODO: This could probably be handled better with a custom type rather than just (string, *string)
func handleLogLine(line string) (string, *string) {
	// The various regexes that match server lines
	playerJoin := regexp.MustCompile(`ServerLobby: New player (.+) with online id [0-9][0-9]?`)
	playerLeave := regexp.MustCompile(`ServerLobby: (.+) disconnected$`)
	noMorePlayers := regexp.MustCompile(`STKHost.+There are now 0 peers\.$`)
	serverStart := regexp.MustCompile(`Listening has been started`)

	// Start the server
	if serverStart.MatchString(line) {
		log.Print("server ready")
		return "READY", nil
	}

	// Player tracking
	if playerJoin.MatchString(line) {
		matches := playerJoin.FindSubmatch([]byte(line))
		player := string(matches[1])
		log.Printf("Player %s joined\n", player)
		return "PLAYERJOIN", &player
	}
	if playerLeave.MatchString(line) {
		matches := playerLeave.FindSubmatch([]byte(line))
		player := string(matches[1])
		log.Printf("Player %s disconnected", player)
		return "PLAYERLEAVE", &player
	}

	// All the players left, send a shutdown
	if noMorePlayers.MatchString(line) {
		log.Print("server has no more players. shutting down")
		return "SHUTDOWN", nil
	}
	return "", nil
}
