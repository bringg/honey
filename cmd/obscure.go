package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rclone/rclone/fs/config/obscure"
)

var obscureCmd = &cobra.Command{
	Use:   "obscure",
	Short: `Obscure password for use in the honey config file.`,
	Long: `In the honey config file, human readable passwords are
obscured. Obscuring them is done by encrypting them and writing them
out in base64. This is **not** a secure way of encrypting these
passwords as honey can decrypt them - it is to prevent "eyedropping"
- namely someone seeing a password in the honey config file by
accident.

Many equally important things (like access tokens) are not obscured in
the config file. However it is very hard to shoulder surf a 64
character hex token.

This command can also accept a password through STDIN instead of an
argument by passing a hyphen as an argument. This will use the first
line of STDIN as the password not including the trailing newline.

echo "secretpassword" | honey obscure -

If there is no data on STDIN to read, honey obscure will default to
obfuscating the hyphen itself.`,
	RunE: func(command *cobra.Command, args []string) error {
		CheckArgs(1, 1, command, args)
		var password string
		fi, _ := os.Stdin.Stat()
		if args[0] == "-" && (fi.Mode()&os.ModeCharDevice) == 0 {
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				password = scanner.Text()
			}
			if err := scanner.Err(); err != nil {
				return err
			}
		} else {
			password = args[0]
		}

		obscured := obscure.MustObscure(password)
		fmt.Println(obscured)

		return nil
	},
}
