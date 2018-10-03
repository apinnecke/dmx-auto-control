// Copyright © 2017 Alexander Pinnecke <alexander.pinnecke@googlemail.com>
//

package artnet

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	root "github.com/StageAutoControl/controller/cmd"
	"github.com/StageAutoControl/controller/cntl"
	"github.com/StageAutoControl/controller/cntl/transport"
	artnetTransport "github.com/StageAutoControl/controller/cntl/transport/artnet"
	"github.com/jsimonetti/go-artnet"
	"github.com/spf13/cobra"
)

const (
	form = "<universe 0-65,536> <channel 0-511> <value 0-255> ..."
)

// Send represents the ArtNetTest command
var Send = &cobra.Command{
	Use:   "send",
	Short: "Send dmx command to a artnet device (shell mode)",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		var ip net.IP
		var err error

		root.Logger.Info("InterfaceName is empty, searching for suitable one ...")
		ip, err = artnetTransport.FindArtNetIP()
		if err != nil {
			root.Logger.Fatal(err)
		}

		root.Logger.Infof("Using interface with IP %s", ip.String())

		if len(ip) == 0 {
			root.Logger.Fatal("No IP found")
		}

		host, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		c := artnet.NewController(host, ip, artnet.NewLogger(root.Logger))
		var wg sync.WaitGroup

		go func() {
			wg.Add(1)
			if err := c.Start(); err != nil {
				root.Logger.Fatalf("Error during sending: %v", err)
			}

			wg.Done()
		}()

		root.Logger.Info("Waiting 5sec for nodes to register")
		time.Sleep(5 * time.Second)

		root.Logger.Infof("Entering interactive mode. Please enter the lines in the form %s", form)
		reader := bufio.NewReader(os.Stdin)
		var universe, channel, value uint64

		state := artnetTransport.NewState()

		for {
			fmt.Print("> ")
			text, _ := reader.ReadString('\n')
			text = strings.Replace(text, "\n", "", -1)

			params := strings.Split(strings.TrimSpace(text), " ")
			if len(params) != 3 {
				root.Logger.Errorf("Please enter the form %s", form)
				continue
			}

			if universe, err = strconv.ParseUint(params[0], 10, 16); err != nil {
				root.Logger.Errorf("Unable to parse universe: %v", err)
				continue
			}
			if channel, err = strconv.ParseUint(params[1], 10, 16); err != nil {
				root.Logger.Errorf("Unable to parse channel: %v", err)
				continue
			}
			if value, err = strconv.ParseUint(params[2], 10, 8); err != nil {
				root.Logger.Errorf("Unable to parse value: %v", err)
				continue
			}

			// root.Logger.Infof("Sending u=%d, c=%d, v=%d", universe, channel, value)

			state.Set(uint16(universe), uint8(channel), uint8(value))

			for u, dmx := range state {
				c.SendDMXToAddress(dmx, transport.UniverseToAddress(cntl.DMXUniverse(u)))
			}
		}

		c.Stop()
		wg.Wait()

		root.Logger.Infof("num: %d", runtime.NumGoroutine())
	},
}

func init() {
	ArtNetCmd.AddCommand(Send)
}
