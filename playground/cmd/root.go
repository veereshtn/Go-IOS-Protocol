package cmd

import (
	"fmt"
	"os"

	"strings"

	"strconv"

	"sort"

	"github.com/iost-official/Go-IOS-Protocol/common"
	"github.com/iost-official/Go-IOS-Protocol/core/state"
	"github.com/iost-official/Go-IOS-Protocol/verifier"
	"github.com/iost-official/Go-IOS-Protocol/vm"
	"github.com/iost-official/Go-IOS-Protocol/vm/lua"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var cfgFile string
var valuesFile string
var language string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "playground",
	Short: "Playground of IOST script",
	Long: `Playground of IOST script, usage:
	playground a.lua b.lua ... --values value.yml
Playground runs lua script by turns.
`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		db := Database{make(map[string][]byte)}
		mdb := state.NewDatabase(&db)
		pool := state.NewPool(mdb)

		m := make(map[interface{}]interface{})

		ctx := vm.BaseContext()

		vf, err := ReadFile(valuesFile)
		if err != nil {
			fmt.Println("no values specified, work as everything is nil")
		} else {
			err = yaml.Unmarshal(vf, &m)
			if err != nil {
				panic(err)
			}

			for k, v := range m {
				switch v.(type) {
				case map[interface{}]interface{}:
					for k2, v2 := range v.(map[interface{}]interface{}) {
						vc, err := state.ParseValue(v2.(string))
						if err != nil {
							panic(err)
						}
						pool.PutHM(state.Key(k.(string)), state.Key(k2.(string)), vc)
					}
				case string:
					vc, err := state.ParseValue(v.(string))
					if err != nil {
						panic(err)
					}
					pool.Put(state.Key(k.(string)), vc)
				}
			}

			ctx = vm.BaseContext()
			ph, err := pool.GetHM("context", "parent-hash")
			if err != nil {
				panic(err)
			}
			wit, err := pool.GetHM("context", "witness")
			if err != nil {
				panic(err)
			}
			height, err := pool.GetHM("context", "height")
			if err != nil {
				panic(err)
			}
			timestamp, err := pool.GetHM("context", "timestamp")
			if err != nil {
				panic(err)
			}

			ctx.ParentHash = common.Base58Decode(ph.(*state.VString).EncodeString()[1:])
			ctx.Witness = vm.IOSTAccount(wit.EncodeString()[1:])
			ctx.BlockHeight = int64(height.(*state.VFloat).ToFloat64())
			ctx.Timestamp = int64(timestamp.(*state.VFloat).ToFloat64())
		}

		v := verifier.NewCacheVerifier()
		v.Context = ctx

		var (
			pool2 state.Pool
			gas   uint64
		)

		pool2 = pool.Copy()

		switch language {
		case "lua":
			for _, file := range args {
				code := ReadSourceFile(file)
				parser, err := lua.NewDocCommentParser(code)
				if err != nil {
					panic(err)
				}
				parser.Debug = true
				sc, err := parser.Parse()
				if err != nil {
					panic(err)
				}

				sc.SetPrefix(file[strings.LastIndex(file, "/")+1 : strings.LastIndex(file, ".")])

				v.StartVM(sc)

				pool2, gas, err = v.Verify(sc, pool2)
				if err != nil {
					fmt.Println("error:", err.Error())
				}
			}
		default:
			fmt.Println(language, "not supported")
		}

		pool2.Flush()
		fmt.Println("======Report")
		fmt.Println("gas spend:", gas)
		fmt.Println("state trasition:")

		var ss []string
		for k, v := range db.Normal {
			if strings.HasPrefix(k, "context.") {
				continue
			}
			var vs string
			val, _ := state.ParseValue(string(v))
			switch val.(type) {
			case *state.VBool:
				vs = "(bool) "
				vs += val.(*state.VBool).EncodeString()
			case *state.VFloat:
				vs = "(float) "
				vs += strconv.FormatFloat(val.(*state.VFloat).ToFloat64(), 'f', 6, 64)
			case *state.VMap:
				vs = "(map) "
				vs += val.EncodeString() + "}"
			case *state.VString:
				vs = "(string) "
				vs += val.EncodeString()[1:]
			}
			ss = append(ss, fmt.Sprintf("  %v >  %v\n", k, vs))
		}
		sort.Strings(ss)
		for _, v := range ss {
			fmt.Print(v)
		}

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVarP(&valuesFile, "values", "v", "values.yaml", "set init values, default ./values.yaml")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "set config default ./values.yaml")
	rootCmd.PersistentFlags().StringVarP(&language, "lang", "l", "lua", "set language of contract, default lua")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("./values.yaml")
	}
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
