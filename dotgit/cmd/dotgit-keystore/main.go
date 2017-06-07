package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bytearena/bytearena/dotgit/config"
	"github.com/bytearena/bytearena/dotgit/database"
	"github.com/bytearena/bytearena/dotgit/protocol"
)

func main() {

	cnf := config.GetConfig()

	f, err := os.OpenFile("/var/log/dotgit-keystore.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

	if len(os.Args) != 2 {
		fmt.Println("Invalid parameters; requires key fingerprint as only parameter")
		f.Close()
		os.Exit(1)
	}

	fingerprint := os.Args[1]
	log.Println("Authenticating key with fingerprint " + fingerprint)

	var db protocol.Database = database.NewGraphQLDatabase()

	err = db.Connect(cnf.GetDatabaseURI())
	if err != nil {
		fmt.Println("Cannot connect to database")
		f.Close()
		os.Exit(1)
	}

	publickey, err := db.FindPublicKeyByFingerprint(fingerprint)
	if err != nil {
		fmt.Println("No key corresponding to given fingerprint")
		f.Close()
		os.Exit(1)
	}

	sshcommand := "/usr/bin/dotgit-ssh " + publickey.Owner.Username
	sshoptions := `no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty,command="` + sshcommand + `"`

	fmt.Println(sshoptions + " " + publickey.Key)
}
