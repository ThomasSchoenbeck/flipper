package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/user"
	"time"

	"github.com/atotto/clipboard"

	color "github.com/fatih/color"
	"github.com/urfave/cli"
	ini "gopkg.in/ini.v1"
)

var (
	myFolder      = "\\.flipper"
	fileName      = "flipper.ini"
	flipperSplash = "Flipper!"
)

// settings
type Setting struct {
	name  string
	value string
}

var ListOfSettings = [2]Setting{
	{name: "setting.prompt", value: "false"},
	{name: "setting.sort", value: "false"}}

var filePath, listName, itemName, itemValue string

type Item struct {
	list  string
	item  string
	value string
}

// terminal colors
var cFlag = color.New(color.FgMagenta).SprintFunc()
var cList = color.New(color.FgYellow).SprintFunc()
var cItem = color.New(color.FgGreen).SprintFunc()
var cValue = color.New(color.FgBlue).SprintFunc()

func setHomeDir() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	filePath = usr.HomeDir + "\\" + myFolder + "\\" + fileName
}

func main() {

	cli.AppHelpTemplate = `NAME:
	 {{.Name}} - {{.Usage}} ({{.Version}})

USAGE:
	 {{.HelpName}} {{if .VisibleFlags}}[flag(s)]{{end}}{{if .Commands}} listname [itemname] [itemvalue]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{end}}
   {{if len .Authors}}
AUTHOR:
	 {{range .Authors}}{{ . }}{{end}}
	 {{end}}{{if .Commands}}
COMMANDS:
	 {{range .Commands}}{{if not .HideHelp}}   {{join .Names ", "}}{{ "\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
GLOBAL OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}{{if .Copyright }}
COPYRIGHT:
   {{.Copyright}}
   {{end}}

`

	setHomeDir()
	cfg, errors := ini.Load(filePath)
	if errors != nil {
		fmt.Println("No file available. creating file", fileName, "in Folder", myFolder)
		f, err := os.Create(filePath)
		if err != nil {
			fmt.Println("Could not create the file", err)
		}
		if err = f.Close(); err != nil {
			fmt.Println("Could not close the file", err)
		}
		os.Exit(0)
	} else {
		// fmt.Println("file loaded") // debug
	}

	readAndWriteSettings(cfg)

	const layout = "2006-01-02 15:04:05"
	// t := time.Now().Format(layout)
	// fmt.Println(t.Format(layout))
	// fmt.Println(t)

	app := cli.NewApp()
	app.Version = "0.1.0"
	app.Compiled = time.Now()
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Thomas Schönbeck",
			Email: "somemailaddress",
		},
	}
	app.Usage = "snippet saver"
	app.UsageText = "save your snippets to clear some brain space"

	// var deleteFlag string

	app.Flags = []cli.Flag{

		cli.BoolFlag{
			Name:  "delete, d",
			Usage: "delete an item or an entire `LIST`",
		},

		// }, cli.StringFlag{
		// 	Name:  "list, l",
		// 	Usage: "print contents of a list",
		// },
		// cli.StringFlag{
		// 	Name:  "all, a",
		// 	Usage: "print contents of a list",
		// },
	}

	app.Action = func(c *cli.Context) error {

		if c.Bool("delete") {

			if c.NArg() > 1 {
				// fmt.Println("im in delete, NArg > 1 (", c.NArg(), ")")
				deleteSomething(cfg, c)
				return nil
			} else if c.NArg() == 1 {
				// fmt.Println("im in delete, NArg == 1")
				listName = c.Args().Get(0)
				deleteList(cfg)
				return nil
			}

		}

		if c.NArg() == 0 {
			// fmt.Println("NArg == 0")

			lists := cfg.SectionStrings()
			if len(lists) > 0 {

				for _, nameOfList := range lists {
					sec, _ := cfg.GetSection(nameOfList)
					items := sec.Keys()
					if nameOfList != "DEFAULT" { // filter out default list (used for values without a list)
						fmt.Fprintln(color.Output, cList(nameOfList), "("+cItem(len(items))+")")
					}
				}

				return nil

			} else {
				showCommandStructure()
				return nil
			}
		} else if c.NArg() == 1 {
			listName = c.Args().Get(0)

			sec, err := cfg.GetSection(listName)
			// fmt.Println(sec)
			// fmt.Println(err)
			if err == nil {
				fmt.Fprintln(color.Output, "list", cList(listName), "exists") // debug
				items := sec.KeysHash()
				if len(items) > 0 {
					// fmt.Fprintln(color.Output, "items in List", cList(list)) // debug
					for item, value := range items {
						fmt.Fprintln(color.Output, cItem(item), "=", cValue(value))
					}
				}
			} else {
				result, err := lookForItem(cfg, itemName)
				if err == nil {
					copyToClipboard(cfg, result.item)
				} else {
					createList(cfg, listName, "noisy")
				}
			}

		} else if c.NArg() == 2 {

			listName = c.Args().Get(0)
			itemName = c.Args().Get(1)

			_, err := cfg.GetSection(listName)
			if err == nil {
				// fmt.Fprintln(color.Output, "list", cList(listName), "exists") // debug
				_, err := cfg.Section(listName).GetKey(itemName)
				if err == nil {
					// fmt.Fprintln(color.Output, "item", cItem(itemName), "exists in List", cList(listName)) // debug
					copyToClipboard(cfg, itemName)
				} else {
					fmt.Fprintln(color.Output, flipperSplash, cItem(itemName), "does not exist in", cList(listName))
				}
			} else {
				fmt.Fprintln(color.Output, flipperSplash, "List", cList(listName), "does not exist. Cannot copy", cItem(itemName), "to clipboard")
			}

			os.Exit(0)

		} else if c.NArg() == 3 {

			listName = c.Args().Get(0)
			itemName = c.Args().Get(1)
			itemValue = c.Args().Get(2)

			_, err := cfg.GetSection(listName)
			if err == nil {
				// fmt.Fprintln(color.Output, "list", cList(listName), "exists") // debug
				createItemOrOverwrite(cfg)
			} else {
				createList(cfg, listName, "silent")
				createItemOrOverwrite(cfg)
			}
			os.Exit(0)
		}

		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func showCommandStructure() {
	fmt.Fprintln(color.Output, "Usage: flipper", cFlag("[flag(s)]"), cList("listname"), cItem("[itemname]"), cValue("[itemvalue]"))
}

func copyToClipboard(cfg *ini.File, item string) {
	// look in every list for item
	result, err := lookForItem(cfg, item)
	if err == nil {
		// fmt.Fprintln(color.Output, flipperSplash, "value", cValue(value), "of item", cItem(item), "copyied to clipboard")
		fmt.Fprintln(color.Output, flipperSplash, "copied", cValue(result.value), "from", cItem(result.item), "to clipboard")
		clipboard.WriteAll(result.value)
	} else {
		fmt.Fprintln(color.Output, flipperSplash, "item", cItem(item), "not found")
	}
}

func lookForItem(cfg *ini.File, searchedItem string) (Item, error) {
	lists := cfg.SectionStrings()
	for _, nameOfList := range lists {
		sec, _ := cfg.GetSection(nameOfList)
		items := sec.KeysHash()
		if len(items) > 0 {
			// fmt.Fprintln(color.Output, "items in List", cList(searchedItem)) // debug
			for item, value := range items {
				if item == searchedItem {
					// fmt.Fprintln(color.Output, flipperSplash, "item", cItem(item), "found") // debug
					return Item{nameOfList, item, value}, nil
					// break
				}
			}
		}
	}
	err := errors.New("Not found")
	return Item{"", "", ""}, err
}

func readAndWriteSettings(cfg *ini.File) {
	// fmt.Println("running readAndWriteSettings") // debug

	fileChanged := false

	sec, _ := cfg.GetSection("")
	settingsFromFile := sec.KeyStrings()

	if len(settingsFromFile) > 0 {
		// fmt.Fprintln(color.Output, "items in List", cList(listName)) // debug

		for i, setting := range ListOfSettings {
			foundInArray := false

			for _, item := range settingsFromFile {
				if item == setting.name {
					ListOfSettings[i].value = cfg.Section("").Key(setting.name).Value()
					// fmt.Fprintln(color.Output, flipperSplash, "found & read setting", setting.name) // debug
					foundInArray = true
					break
				}
			}

			if !foundInArray {
				_, err := cfg.Section("").NewKey(setting.name, setting.value)
				// fmt.Fprintln(color.Output, flipperSplash, setting.name, "not found. writing to file") // debug
				if err != nil {
					fmt.Fprintln(color.Output, flipperSplash, "could not write", setting.name, err)
				} else {
					fileChanged = true
				}
			}

		}

	} else {
		fmt.Println("settings empty, writing new") // debug
		for _, setting := range ListOfSettings {
			fmt.Println(setting.name, "=", setting.value)
			_, err := cfg.Section("").NewKey(setting.name, setting.value)
			if err != nil {
				fmt.Fprintln(color.Output, flipperSplash, "could not write", setting.name, err)
			} else {
				fileChanged = true
			}

		}
	}

	if fileChanged {
		writeFile(cfg)
	}

}

func deleteSomething(cfg *ini.File, c *cli.Context) {
	// fmt.Println("running processDeleteFlag") // debug
	// got one argument beside the flag, delete list (or item if list does not exist)
	// if len(flag.Args()) == 2 {
	if c.NArg() == 2 {

		listName = c.Args().Get(0)
		itemName = c.Args().Get(1)

		_, err := cfg.GetSection(listName)
		if err == nil {
			if deleteList(cfg) == nil {
				// os.Exit(0)
				return
			} else {
				var list = ""
				var item = listName
				deleteItem(cfg, list, item)
				// os.Exit(0)
				return
			}
		} else {
			// since the list was not found, check if we can delete an item with that name
			var item = listName
			result, err := lookForItem(cfg, item)
			if err == nil {
				deleteItem(cfg, result.list, result.item)
				// cfg.Section(result.list).DeleteKey(result.item)
				// fmt.Fprintln(color.Output, flipperSplash, "deleted", cItem(result.item), "from", cList(result.list))
				// os.Exit(0)
				return
			} else {
				fmt.Fprintln(color.Output, flipperSplash, "List", cList(listName), "does not exist")
				// os.Exit(0)
				return
			}

		}

		// got two arguments beside the flag, delete item
	} else if c.NArg() == 3 {

	}
}

func createList(cfg *ini.File, list string, silent string) {
	cfg.NewSection(list)
	if silent != "silent" {
		fmt.Fprintln(color.Output, flipperSplash, "list", cList(list), "created")
	}
	cfg.SaveTo(filePath)
}

func createItemOrOverwrite(cfg *ini.File) {
	key, err := cfg.Section(listName).GetKey(itemName)
	if err == nil {
		// fmt.Fprintln(color.Output, "item", cItem(itemName), "exists in List", cList(listName)) // debug
		overwriteValue(cfg, key)
	} else {
		addItemToList(cfg)
	}
}

func addItemToList(cfg *ini.File) {
	_, err := cfg.Section(listName).NewKey(itemName, itemValue)
	if err == nil {
		fmt.Fprintln(color.Output, flipperSplash, "added", cItem(itemName), "with value", cValue(itemValue), "to List", cList(listName))
	}

	writeFile(cfg)
}

func overwriteValue(cfg *ini.File, key *ini.Key) {
	key.SetValue(itemValue)
	fmt.Fprintln(color.Output, flipperSplash, cItem(itemName), "overwritten with value", cValue(itemValue), "in List", cList(listName))
	cfg.SaveTo(filePath)
}

func deleteList(cfg *ini.File) error {
	// fmt.Println("running deleteList") // debug

	listExists := false
	lists := cfg.SectionStrings()
	for _, nameOfList := range lists {
		// fmt.Println("list", listName, "array list", nameOfList)
		if nameOfList == listName {

			listExists = true
			promptUser(listName)
			cfg.DeleteSection(listName)
			fmt.Fprintln(color.Output, flipperSplash, "List", cList(listName), "deleted")
			writeFile(cfg)
		}
	}
	if !listExists {
		fmt.Fprintln(color.Output, flipperSplash, "List", cList(listName), "does not exist")
		return fmt.Errorf("List %q not found", listName)
	}
	return nil
}

func promptUser(item string) {
	for _, setting := range ListOfSettings {
		if setting.name == "setting.prompt" {
			if setting.value == "true" {
				fmt.Println("do you really want to delete", item+"?")
			}
		}
	}
}

func deleteItem(cfg *ini.File, list string, item string) {
	fmt.Println("running deleteItem") // debug
	keyExists := false

	if list != "" {
		keyExists = cfg.Section(list).HasKey(item)
	} else {
		lists := cfg.SectionStrings()
		for _, nameOfList := range lists {
			keyExists := cfg.Section(nameOfList).HasKey(item)
			if keyExists {
				list = nameOfList
				break
			}
		}
	}

	if keyExists {
		promptUser(item)
		cfg.Section(list).DeleteKey(item)
		fmt.Fprintln(color.Output, flipperSplash, "delted", cItem(item), "from List", cList(list))
		writeFile(cfg)
	} else {
		if list != "" {
			fmt.Fprintln(color.Output, flipperSplash, "item", cItem(item), "not found in List", cList(list))
		} else {
			fmt.Fprintln(color.Output, flipperSplash, "item", cItem(item), "not found in any List")
		}
	}

}

func writeFile(cfg *ini.File) {
	err := cfg.SaveTo(filePath)
	if err != nil {
		fmt.Println("cannot write file")
	}
}
