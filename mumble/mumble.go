package mumble

import (
	"log"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"layeh.com/gumble/gumble"
	"github.com/agnivade/levenshtein"
)

// Kill marks a player as dead
func Kill(c *gumble.Client, player string, gamestate string, deadplayers []string) []string {
	alive := c.Channels.Find("AmongUs", "Alive")

	log.Println("In game player:", player)
	aliveusers := c.Channels[alive.ID].Users
	var player2 string
	deadMumbleUser := FindUserForPlayer(aliveusers, player)
	if deadMumbleUser == nil {
		log.Println("Mumble user is unknown for dead player", player2)
	}
	player2 = deadMumbleUser.Name

	duplicateplayer := 0

	for _, s := range deadplayers {
		if s == player2 {
			duplicateplayer = duplicateplayer + 1
		}
	}

	if player2 != "" {
		if duplicateplayer == 0 {
			log.Println(player2, "is now dead")
			deadplayers = append(deadplayers, strings.TrimSpace(player2))
		} else {
			log.Println(player2, "is already dead")
		}
	} else {
		log.Println("Ignoring blank player name")
	}

	log.Println("Dead Players:", deadplayers)

	return deadplayers
}

// Startgame starts the game
func Startgame(c *gumble.Client) {
	lobby := c.Channels.Find("AmongUs", "Lobby")
	alive := c.Channels.Find("AmongUs", "Alive")
	lobbyusers := c.Channels[lobby.ID].Users
	for _, element := range lobbyusers {
		element.Move(alive)
		element.SetMuted(true)
		element.SetDeafened(true)
		log.Println("Moving", element.Name, "to #alive")
	}
}

// Meeting starts discussion phase
func Meeting(c *gumble.Client, deadplayers []string) {
	alive := c.Channels.Find("AmongUs", "Alive")
	aliveusers := c.Channels[alive.ID].Users

	for _, element := range aliveusers {
		log.Println("Unmute player", element.Name)
		element.SetMuted(false)
		element.SetDeafened(false)
		log.Println(element.Name, "is alive")
	}

	for _, deadplayer := range deadplayers {
		user := c.Users.Find(deadplayer)
		log.Println("Mute player", user.Name)
		user.SetMuted(true)
		user.SetDeafened(false)
		user.Move(alive)
		log.Println(user.Name, "is dead")
	}
}

// Resumegame Resumes Game (tasks)
func Resumegame(c *gumble.Client, deadplayers []string) {
	alive := c.Channels.Find("AmongUs", "Alive")
	dead := c.Channels.Find("AmongUs", "Dead")

	aliveusers := c.Channels[alive.ID].Users

	log.Println("Resuming game")

	for _, element := range aliveusers {
		log.Println("Mute player", element.Name)
		element.SetMuted(true)
		element.SetDeafened(true)
		log.Println(element.Name, "is alive")
	}

	for _, deadplayer := range deadplayers {
		user := c.Users.Find(deadplayer)
		log.Println("Unmute player", user.Name)
		user.SetMuted(false)
		user.SetDeafened(false)
		user.Move(dead)
		log.Println(user.Name, "is dead")
	}
}

// Endgame ends game
func Endgame(c *gumble.Client) {
	lobby := c.Channels.Find("AmongUs", "Lobby")
	alive := c.Channels.Find("AmongUs", "Alive")
	dead := c.Channels.Find("AmongUs", "Dead")

	aliveusers := c.Channels[alive.ID].Users
	deadusers := c.Channels[dead.ID].Users

	for _, element := range aliveusers {
		element.Move(lobby)
		element.SetMuted(false)
		element.SetDeafened(false)
		log.Println("Unmute player", element.Name)
	}

	for _, element := range deadusers {
		element.Move(lobby)
		element.SetMuted(false)
		element.SetDeafened(false)
		log.Println("Unmute player", element.Name)
	}
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}

func FindUserForPlayer(users gumble.Users, player string) *gumble.User {
	player = strings.ToLower(player)
	// log.Println("Resolving user: ", player)

	// Fast match: by extact same nickname
	foundUser := users.Find(player)
	if foundUser != nil {
		// log.Println("Matching user: ", foundUser.Name)
		return foundUser
	}

	var mumbleUserName string
	
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	
	bestMatchingDist := 9001
	var bestMatchingUser *gumble.User

	for _, mumbleUser := range users {
		// Fast match: if comment contains IGN
		if mumbleUser.Comment == player {
			return mumbleUser
		}

		mumbleUserName = strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsNumber(r) {
				return r
		}
			return -1
		}, mumbleUser.Name)
		// log.Println("Mumble user:", mumbleUserName)
		// log.Println("Mumble user cmt:", mumbleUser.Comment)
		mumbleUserName, _, _ := transform.String(t, mumbleUserName)
		mumbleUserName = strings.ToLower(mumbleUserName)
		// log.Println("Filtered mumble user:", mumbleUserName)

		levDist := levenshtein.ComputeDistance(mumbleUserName, player)
		// log.Println("Lev dist:", levDist)

		if (levDist < bestMatchingDist) {
			bestMatchingUser = mumbleUser
			bestMatchingDist = levDist
		}
	}

	// log.Println("Best dist:", bestMatchingDist)

	if (bestMatchingDist > 9000) || (bestMatchingDist > (len(player) - 2)) {
		// log.Println("No best match")
		return nil
	}

	return bestMatchingUser
}

// Namecheck makes sure player has a valid comment
func Namecheck(c *gumble.Client, player string) bool {
	//var player2 string

	lobby := c.Channels.Find("AmongUs", "Lobby")
	lobbyusers := c.Channels[lobby.ID].Users

	log.Println("Checking if", player, "has a mumble user set")
	mumbleUser := FindUserForPlayer(lobbyusers, player)
	if mumbleUser != nil {
		log.Println("User set:", mumbleUser.Name, "==", player)
		return true
	}

	log.Println("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	log.Println("Player", player, "does not have a mumble user set.")
	log.Println("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	return false
}
