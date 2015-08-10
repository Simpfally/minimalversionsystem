package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	//"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

type DB struct {
	filename   string
	dirname    string
	CommitList map[string]bool
	ID         int
	file       *os.File
}

func LoadDB(filename, dirname string) (*DB, error) {
	er := ger("Loading DB")
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, er("Opening file", err)
	}
	dec := gob.NewDecoder(file)
	db := &DB{}
	stats, err := file.Stat()
	if err != nil {
		return nil, er("Getting file stats", err)
	}

	if stats.Size() > 0 {
		err = dec.Decode(db)
		if err != nil {
			return nil, er("Decoding non empty db", err)
		}

	} else {
		db = &DB{}
		db.CommitList = make(map[string]bool)
	}
	db.filename = filename
	db.dirname = dirname
	db.file = file
	return db, nil
}

func (db *DB) Write() error {
	er := ger("Writing DB")
	_, err := db.file.Seek(0, 0)
	if err != nil {
		return er("File seek", err)
	}
	enc := gob.NewEncoder(db.file)
	err = enc.Encode(db)
	if err != nil {
		return er("Encoding", err)
	}
	err = db.file.Close()
	if err != nil {
		return er("Closing file", err)
	}
	return nil
}

func (db *DB) Add(name string) error {
	if _, ok := db.CommitList[name]; ok {
		return fmt.Errorf("%s already in the list", name)
	}
	db.CommitList[name] = true
	return nil
}

func (db *DB) Rem(name string) error {
	if _, ok := db.CommitList[name]; !ok {
		return fmt.Errorf("%s wasn't in the list", name)
	}
	delete(db.CommitList, name)
	return nil
}
func (db *DB) Com() error {
	er := ger("")
	err := os.MkdirAll(db.dirname, os.ModeDir|0777)
	db.ID++
	if err != nil {
		return er("Mkdir", err)
	}
	for name := range db.CommitList {
		filea, err := os.Open(name)
		if err != nil {
			return er("Opening committed file", err)
		}
		defer filea.Close()

		content, err := ioutil.ReadAll(filea)
		if err != nil {
			return er("Reading committed file", err)
		}

		filename := db.name(name, db.ID)
		fileb, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			return err
		}
		defer fileb.Close()
		_, err = fileb.Write(content)
		if err != nil {
			return err
		}
		filea.Close()
		fileb.Close()
	}
	return nil
}

func (db *DB) name(filename string, id int) string {
	return fmt.Sprintf("%s/%s_%d.min", db.dirname, filename, id)
}

func (db *DB) Rev(id int) error {
	er := ger("")
	if len(db.CommitList) == 0 {
		return er("Nothing yet", nil)
	}
	for name := range db.CommitList {
		filea, err := os.Open(db.name(name, id))
		if err != nil {
			continue
		}
		defer filea.Close()
		for i := id + 1; i <= db.ID; i++ {
			fmt.Println("deleting", db.name(name, i))
			err := os.Remove(db.name(name, i))
			if err != nil {
				fmt.Println(err)
			}
		}

		content, err := ioutil.ReadAll(filea)
		if err != nil {
			return er("Reading committed file", err)
		}

		fileb, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}
		defer fileb.Close()
		_, err = fileb.Write(content)
		if err != nil {
			return err
		}
	}
	db.ID -= id
	return nil
}

func (db *DB) List() error {
	i := 0
	fmt.Println("Last rev :", db.ID)
	for name := range db.CommitList {
		fmt.Printf("%d - %s\n", i, name)
		i++
	}
	return nil
}

var missArg error = errors.New("Wrong number of argument")
var wrongCmd error = errors.New("Wrong command")

func HandleCmd(db *DB) error {
	er := ger("Handling cmd")
	numArgs := len(os.Args) - 1
	cmd := os.Args[1]
	var name string
	if numArgs == 2 {
		name = os.Args[2]
	}

	switch cmd {
	case "add":
		if numArgs != 2 {
			return missArg
		}
		err := db.Add(name)
		if err != nil {
			return er("Add", err)
		}
		return db.Write()
	case "rem":
		if numArgs != 2 {
			return missArg

		}
		err := db.Rem(name)
		if err != nil {
			return er("Rem", err)
		}

		return db.Write()
	case "com":

		err := db.Com()
		if err != nil {
			return er("Com", err)
		}

		return db.Write()
	case "rev":
		id, err := strconv.Atoi(name)
		if err != nil {
			return er("Rev", err)
		}
		return db.Rev(id)
	case "list":
		return db.List()
	default:
		return wrongCmd
	}
	return nil
}

func main() {
	numArgs := len(os.Args) - 1
	if numArgs < 1 || numArgs > 2 {
		printHelp()
		os.Exit(0)
	}
	filename := "commitfile"
	dirname := "commitdir"
	db, err := LoadDB(filename, dirname)
	if err != nil {
		log.Fatal(err)
	}
	defer db.file.Close()

	err = HandleCmd(db)
	if err == missArg || err == wrongCmd {
		fmt.Println(err)
		printHelp()
	} else if err != nil {
		log.Println(err)
	}
}

func printHelp() {
	fmt.Printf("MVCS - help - Commands :\n" +
		"\tadd [filename]\n" +
		"\trem [filename]\n" +
		"\tcom\n" +
		"\trev [rev numb]\n" +
		"\tlist\n")
}

func ger(ctxt string) func(string, error) error {
	return func(msg string, err error) error {
		return fmt.Errorf("%s : %s : %v", ctxt, msg, err)
	}
}
