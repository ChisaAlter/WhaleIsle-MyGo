package main

import (
	"fmt"
	"os"
	"path/filepath"

	"whaleisle/internal/desktop/components"
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/paths"
)

func main() {
	fmt.Println("UserData", paths.UserDataDir())
	fmt.Println("Electron", importer.ElectronUserDataDir())
	for _, root := range []string{paths.UserDataDir(), importer.ElectronUserDataDir()} {
		rows := components.ListUserData(root)
		fmt.Println("root", root, "n", len(rows))
		for _, r := range rows {
			fmt.Printf("  id=%s name=%s state=%s ver=%s inst=%s\n", r.ID, r.Name, r.State, r.Version, r.InstalledVersion)
		}
		ents, err := os.ReadDir(filepath.Join(root, "components"))
		fmt.Println(" readdir", err)
		for _, e := range ents {
			fmt.Println("  ", e.Name())
		}
	}
}
