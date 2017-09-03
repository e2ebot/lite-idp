// Copyright © 2017 Aaron Donovan <amdonov@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/amdonov/lite-idp/idp"
	"github.com/gorilla/handlers"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "lite-idp",
	Short: "SAML 2 Identity Provider",
	Long:  `TODO`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Listen for shutdown signal
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt)
		idp := &idp.IDP{}
		handler, err := idp.Handler()
		if err != nil {
			return err
		}
		server := &http.Server{
			TLSConfig: idp.TLSConfig,
			Handler:   handlers.CombinedLoggingHandler(os.Stdout, hsts(handler)),
			Addr:      viper.GetString("listen-address"),
		}
		go func() {
			// Handle shutdown signal
			<-stop
			server.Shutdown(context.Background())
		}()

		log.Infof("listening for connections on %s", server.Addr)
		if err = server.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
			return err
		}
		log.Info("server shutdown cleanly")
		return nil
	},
}

type hstsHandler struct {
	handler http.Handler
}

func (h *hstsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	h.handler.ServeHTTP(w, r)
}

func hsts(h http.Handler) http.Handler {
	return &hstsHandler{h}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "/etc/lite-idp/lite-idp.yaml", "config file")
	RootCmd.Flags().StringP("tls-certificate", "c", "/etc/lite-idp/cert.pem", "PEM encoded certificate file")
	RootCmd.Flags().StringP("tls-private-key", "k", "/etc/lite-idp/key.pem", "PEM encoded private key file")
	RootCmd.Flags().String("tls-ca", "", "PEM encoded file containing trusted certificate authorities (default \"OS trusted authorities\")")
	RootCmd.Flags().String("listen-address", "127.0.0.1:9443", "host:port to listen for connections")
	RootCmd.Flags().String("server-name", "idp.example.com:9443", "FQDN used and optional port used to construct URLs")
	RootCmd.Flags().String("entity-id", "", "SAML entityID (default \"https://$SERVER_NAME/\")")
	RootCmd.Flags().String("metadata-path", "/metadata", "server path for serving metadata")
	RootCmd.Flags().String("sso-service-path", "/SAML2/Redirect/SSO", "server path for redirect-based SSO service")
	RootCmd.Flags().String("artifact-service-path", "/SAML2/SOAP/ArtifactResolution", "server path for artifact resolution service")
	RootCmd.Flags().String("attribute-service-path", "/SAML2/SOAP/AttributeQuery", "server path for attribute query service")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("/etc/lite-idp")
		viper.SetConfigName("lite-idp")
	}

	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.BindPFlags(RootCmd.Flags())

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("using config file:", viper.ConfigFileUsed())
	} else {
		fmt.Println("failed to load config file:", err)
	}
}
