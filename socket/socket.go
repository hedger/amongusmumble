package socket

import (
	"amongusmumble/mumble"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	socketio "github.com/googollee/go-socket.io"
	"layeh.com/gumble/gumble"
)

/*
    public enum PlayerAction
    {
        Joined,         // 0
        Left,           // 1
        Died,           // 2
        ChangedColor,   // 3
        ForceUpdated,   // 4
        Disconnected,   // 5
        Exiled          // 6
    }
*/

type player struct {
    Action       int    `json:"Action"`
	Name         string `json:"Name"`
	Color        int    `json:"Color"`
	IsDead       bool   `json:"IsDead"`
	Disconnected bool   `json:"Disconnected"`
}

// SocketioServer Listner to capture
func SocketioServer(client *gumble.Client, listenaddress string, listenport string) {
	var deadplayers []string
	var gamestate string
	var gameup bool
	var gamestatetime time.Time

	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		log.Println("connected:", s.ID())
		return nil
	})

	server.OnEvent("/", "connect", func(s socketio.Conn, msg string) {
		log.Println("set connect code:", msg)
		s.Emit("reply", "set guildID successfully")
	})
	server.OnEvent("/", "state", func(s socketio.Conn, msg string) {
		log.Println("Phase received from capture: ", msg)

		switch msg {
		case "0":
			gamestate = "LOBBY"
		case "1":
			gamestate = "TASKS"
		case "2":
			gamestate = "DISCUSSION"
		}
		log.Println("Gamestate set:", gamestate)
		switch gamestate {
		case "MENU":
			log.Println("Gamemode: Menu")
			mumble.Endgame(client)
			deadplayers = nil
			gameup = false
		case "LOBBY":
			log.Println("Gamemode: LOBBY")
			mumble.Endgame(client)
			deadplayers = nil
			gameup = false
		case "DISCUSSION":
			log.Println("Gamemode: DISCUSSION")
			mumble.Meeting(client, deadplayers)
		case "TASKS":
			log.Println("Gamemode: TASKS")
			gamestatetime = time.Now()
			log.Println("Game State Time:", gamestatetime)
			time.Sleep(5 * time.Second)
			if gameup == false {
				mumble.Startgame(client)
			} else {
				mumble.Resumegame(client, deadplayers)
			}
			gameup = true
		}
	})
	server.OnEvent("/", "player", func(s socketio.Conn, msg string) {
		log.Println("Player received from capture: ", msg)
		log.Println("Gamestate: ", gamestate)
		player := player{}
		_ = json.Unmarshal([]byte(msg), &player)

		if gamestate == "LOBBY" {
            if player.Action != 1 /* did not just left */ && player.Action != 5 /* did not just disconnected */ {
    			isFound := mumble.Namecheck(client, strings.TrimSpace(player.Name))
                if isFound == false {
					channel := client.Channels.Find("AmongUs")
                    channel.Send("'" + player.Name + "' is invalid", true)
                }
            }
		} else {
			if player.IsDead == true {
				deadplayers = mumble.Kill(client, strings.TrimSpace(player.Name), gamestate, deadplayers)
				duration := time.Since(gamestatetime)
				if duration.Seconds() < 10 {
					log.Println("Move", player.Name, "to Dead now")
					alive := client.Channels.Find("AmongUs", "Alive")
					dead := client.Channels.Find("AmongUs", "Dead")
					log.Println("In game player:", player)
					aliveusers := client.Channels[alive.ID].Users

					mumbleUser := mumble.FindUserForPlayer(aliveusers, player.Name)
					if mumbleUser == nil {
						log.Println(player.Name, "Cannot find mumble user!!!")
					} else {
						mumbleUser.Move(dead)
						mumbleUser.SetMuted(false)
						mumbleUser.SetDeafened(false)
						log.Println(player.Name, "Moved to Dead")
					}

				} else {
					log.Println("Move", player.Name, "to Dead at end of round")
				}
			}
		}
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		log.Println("meet error:", e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Println("closed", reason)
	})

	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)
	log.Println("Serving at", listenaddress, ":", listenport, "...")
	log.Fatal(http.ListenAndServe(listenaddress+":"+listenport, nil))
}
