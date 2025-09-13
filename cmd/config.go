package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
)

var server string
var username string
var password string

type JSONConfig struct {
	Server   string `json:"server"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func ReadConfigJSONFile(jsonConfig *JSONConfig) {
	jsonFile, err := os.Open("config.json")
	if err != nil {
		fmt.Println(err)
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &jsonConfig)
}

func writeConfigJSONFile(server, username, password string) error {
	jsonFile := JSONConfig{Server: server, Username: username, Password: password}
	jsonData, err := json.MarshalIndent(jsonFile, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile("./config.json", jsonData, 0644)
	if err != nil {
		return err
	}
	return nil
}

func printMissingFlag(flag string) {
	fmt.Printf("Error: %s is required\n", flag)
	fmt.Printf("Usage: kern config --%s <value>", flag)
}

func ExitIfIsMissingFields() {
	if CONFIG.Server == "" || CONFIG.Username == "" || CONFIG.Password == "" {
		fmt.Println("[error] is missing important fields on CONFIG, do `kern config` again")
		fmt.Println(`- server: ` + CONFIG.Server)
		fmt.Println(`- username: ` + CONFIG.Username)
		fmt.Println(`- password: ` + CONFIG.Password)
		os.Exit(1)
	}
}

var configCommand = &cobra.Command{
	Use:   "config",
	Short: "Set up server for monitoring",
	Run: func(cmd *cobra.Command, args []string) {
		if server == "" {
			printMissingFlag("server")
			return
		}
		if username == "" {
			printMissingFlag("username")
			return
		}
		if password == "" {
			printMissingFlag("password")
			return
		}

		err := writeConfigJSONFile(server, username, password)
		if err != nil {
			fmt.Println("occurred an erron trying to write kern config.json")
		}
	},
}

func init() {
	rootCmd.AddCommand(configCommand)
	configCommand.Flags().StringVarP(&server, "server", "s", "", "")
	configCommand.Flags().StringVarP(&username, "username", "u", "", "")
	configCommand.Flags().StringVarP(&password, "password", "p", "", "")
}
