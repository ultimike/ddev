package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/version"
	"github.com/mitchellh/go-homedir"
	"os"
	"strings"

	"path/filepath"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// Define flags for the config command
var (
	// createDocroot will allow a nonexistent docroot to be created
	createDocroot bool

	// docrootRelPathArg is the relative path to the docroot where index.php is.
	docrootRelPathArg string

	// projectNameArg is the name of the site.
	projectNameArg string

	// projectTypeArg is the ddev app type, like drupal7/drupal8/wordpress.
	projectTypeArg string

	// phpVersionArg overrides the default version of PHP to be used in the web container, like 5.6/7.0/7.1/7.2/7.3.
	phpVersionArg string

	// httpPortArg overrides the default HTTP port (80).
	httpPortArg string

	// httpsPortArg overrides the default HTTPS port (443).
	httpsPortArg string

	// xdebugEnabledArg allows a user to enable XDebug from a command flag.
	xdebugEnabledArg bool

	// additionalHostnamesArg allows a user to provide a comma-delimited list of hostnames from a command flag.
	additionalHostnamesArg string

	// additionalFQDNsArg allows a user to provide a comma-delimited list of FQDNs from a command flag.
	additionalFQDNsArg string

	// showConfigLocation, if set, causes the command to show the config location.
	showConfigLocation bool

	// uploadDirArg allows a user to set the project's upload directory, the destination directory for import-files.
	uploadDirArg string

	// webserverTypeArgs allows a user to set the project's webserver type
	webserverTypeArg string

	// webImageArg allows a user to set the project's web server container image
	webImageArg string

	// webImageDefaultArg allows a user to unset the specific web server container image
	webImageDefaultArg bool

	// dbImageArg allows a user to set the project's db server container image
	dbImageArg string

	// dbImageDefaultArg allows a user to uset the specific db server container image
	dbImageDefaultArg bool

	// dbaImageArg allows a user to set the project's dba container image
	dbaImageArg string

	// dbaImageDefaultArg allows a user to unset the specific dba container image
	dbaImageDefaultArg bool

	// imageDefaultsArg allows a user to unset all specific container images
	imageDefaultsArg bool

	// webWorkingDirArg allows a user to define the working directory for the web service
	webWorkingDirArg string

	// webWorkingDirDefaultArg allows a user to unset a web service working directory override
	webWorkingDirDefaultArg bool

	// dbWorkingDirArg allows a user to define the working directory for the db service
	dbWorkingDirArg string

	// defaultDbaWorkingDirArg allows a user to unset a db service working directory override
	dbWorkingDirDefaultArg bool

	// dbaWorkingDirArg allows a user to define the working directory for the dba service
	dbaWorkingDirArg string

	// dbaWorkingDirDefaultArg allows a user to unset a dba service working directory override
	dbaWorkingDirDefaultArg bool

	// workingDirDefaultsArg allows a user to unset all service working directory overrides
	workingDirDefaultsArg bool

	// omitContainersArg allows user to determine value of omit_containers
	omitContainersArg string

	// mariadbVersionArg is mariadb version 10.1 or 10.2
	mariaDBVersionArg string

	// nfsMountEnabled sets nfs_mount_enabled
	nfsMountEnabled bool

	// hostDBPortArg sets host_db_port
	hostDBPortArg string

	// hostWebserverPortArg sets host_webserver_port
	hostWebserverPortArg string

	// hostHTTPSPortArg sets host_https_port
	hostHTTPSPortArg string

	// mailhogPortArg is arg for mailhog port
	mailhogPortArg string

	// phpMyAdminPortArg is arg for phpmyadmin container port access
	phpMyAdminPortArg string

	// webImageExtraPackages and dbImageExtraPackages are comma-delimited
	// lists of Debian packages to be added to related containers on build
	webimageExtraPackages string
	dbimageExtraPackages  string

	// projectTLDArg specifies a project top-level-domain; defaults to ddevapp.DdevDefaultTLD
	projectTLDArg string

	// useDNSWhenPossibleArg specifies
	useDNSWhenPossibleArg bool
)

var providerName = ddevapp.ProviderDefault

// extraFlagsHandlingFunc does specific handling for additional flags, and is different per provider.
var extraFlagsHandlingFunc func(cmd *cobra.Command, args []string, app *ddevapp.DdevApp) error

// ConfigCommand represents the `ddev config` command
var ConfigCommand *cobra.Command = &cobra.Command{
	Use:     "config [provider or 'global']",
	Short:   "Create or modify a ddev project configuration in the current directory",
	Example: `"ddev config" or "ddev config --docroot=web  --project-type=drupal8"`,
	Args:    cobra.ExactArgs(0),
	Run:     handleConfigRun,
}

// handleConfigRun handles all the flag processing for any provider
func handleConfigRun(cmd *cobra.Command, args []string) {
	app, err := getConfigApp(providerName)
	if err != nil {
		util.Failed(err.Error())
	}

	homeDir, _ := homedir.Dir()
	if app.AppRoot == filepath.Dir(globalconfig.GetGlobalDdevDir()) || app.AppRoot == homeDir {
		util.Failed("Please do not use `ddev config` in your home directory")
	}

	if cmd.Flags().NFlag() == 0 {
		err = app.PromptForConfig()
		if err != nil {
			util.Failed("There was a problem configuring your project: %v", err)
		}
	} else {
		err = handleMainConfigArgs(cmd, args, app)
		if err != nil {
			util.Failed(err.Error())
		}
		if extraFlagsHandlingFunc != nil {
			err = extraFlagsHandlingFunc(cmd, args, app)
			if err != nil {
				util.Failed("failed to handle per-provider extra flags: %v", err)
			}
		}
	}

	provider, err := app.GetProvider()
	if err != nil {
		util.Failed("Failed to get provider: %v", err)
	}
	err = provider.Validate()
	if err != nil {
		util.Failed("Failed to validate project name %v: %v", app.Name, err)
	}

	err = app.WriteConfig()
	if err != nil {
		util.Failed("Failed to write config: %v", err)
	}

	_, err = app.CreateSettingsFile()
	if err != nil {
		util.Warning("Could not write settings file: %v", err)
	}

	err = provider.Write(app.GetConfigPath("import.yaml"))
	if err != nil {
		util.Failed("Failed to write provider config: %v", err)
	}

	util.Success("Configuration complete. You may now run 'ddev start'.")
}

func init() {
	var err error

	validAppTypes := strings.Join(ddevapp.GetValidAppTypes(), ", ")
	projectTypeUsage := fmt.Sprintf("Provide the project type (one of %s). This is autodetected and this flag is necessary only to override the detection.", validAppTypes)
	projectNameUsage := fmt.Sprintf("Provide the project name of project to configure (normally the same as the last part of directory name)")

	ConfigCommand.Flags().StringVar(&projectNameArg, "project-name", "", projectNameUsage)
	ConfigCommand.Flags().StringVar(&docrootRelPathArg, "docroot", "", "Provide the relative docroot of the project, like 'docroot' or 'htdocs' or 'web', defaults to empty, the current directory")
	ConfigCommand.Flags().StringVar(&projectTypeArg, "project-type", "", projectTypeUsage)
	ConfigCommand.Flags().StringVar(&phpVersionArg, "php-version", "", "The version of PHP that will be enabled in the web container")
	ConfigCommand.Flags().StringVar(&httpPortArg, "http-port", "", "The router HTTP port for this project")
	ConfigCommand.Flags().StringVar(&httpsPortArg, "https-port", "", "The router HTTPS port for this project")
	ConfigCommand.Flags().BoolVar(&xdebugEnabledArg, "xdebug-enabled", false, "Whether or not XDebug is enabled in the web container")
	ConfigCommand.Flags().StringVar(&additionalHostnamesArg, "additional-hostnames", "", "A comma-delimited list of hostnames for the project")
	ConfigCommand.Flags().StringVar(&additionalFQDNsArg, "additional-fqdns", "", "A comma-delimited list of FQDNs for the project")
	ConfigCommand.Flags().StringVar(&omitContainersArg, "omit-containers", "", "A comma-delimited list of container types that should not be started when the project is started")
	ConfigCommand.Flags().BoolVar(&createDocroot, "create-docroot", false, "Prompts ddev to create the docroot if it doesn't exist")
	ConfigCommand.Flags().BoolVar(&showConfigLocation, "show-config-location", false, "Output the location of the config.yaml file if it exists, or error that it doesn't exist.")
	ConfigCommand.Flags().StringVar(&uploadDirArg, "upload-dir", "", "Sets the project's upload directory, the destination directory of the import-files command.")
	ConfigCommand.Flags().StringVar(&webserverTypeArg, "webserver-type", "", "Sets the project's desired webserver type: nginx-fpm, apache-fpm, or apache-cgi")
	ConfigCommand.Flags().StringVar(&webImageArg, "web-image", "", "Sets the web container image")
	ConfigCommand.Flags().BoolVar(&webImageDefaultArg, "web-image-default", false, "Sets the default web container image for this ddev version")
	ConfigCommand.Flags().StringVar(&dbImageArg, "db-image", "", "Sets the db container image")
	ConfigCommand.Flags().BoolVar(&dbImageDefaultArg, "db-image-default", false, "Sets the default db container image for this ddev version")
	ConfigCommand.Flags().StringVar(&dbaImageArg, "dba-image", "", "Sets the dba container image")
	ConfigCommand.Flags().BoolVar(&dbaImageDefaultArg, "dba-image-default", false, "Sets the default dba container image for this ddev version")
	ConfigCommand.Flags().BoolVar(&imageDefaultsArg, "image-defaults", false, "Sets the default web, db, and dba container images")
	ConfigCommand.Flags().StringVar(&webWorkingDirArg, "web-working-dir", "", "Overrides the default working directory for the web service")
	ConfigCommand.Flags().StringVar(&dbWorkingDirArg, "db-working-dir", "", "Overrides the default working directory for the db service")
	ConfigCommand.Flags().StringVar(&dbaWorkingDirArg, "dba-working-dir", "", "Overrides the default working directory for the dba service")
	ConfigCommand.Flags().BoolVar(&webWorkingDirDefaultArg, "web-working-dir-default", false, "Unsets a web service working directory override")
	ConfigCommand.Flags().BoolVar(&dbWorkingDirDefaultArg, "db-working-dir-default", false, "Unsets a db service working directory override")
	ConfigCommand.Flags().BoolVar(&dbaWorkingDirDefaultArg, "dba-working-dir-default", false, "Unsets a dba service working directory override")
	ConfigCommand.Flags().BoolVar(&workingDirDefaultsArg, "working-dir-defaults", false, "Unsets all service working directory overrides")
	ConfigCommand.Flags().StringVar(&mariaDBVersionArg, "mariadb-version", "10.2", "mariadb version to use")
	ConfigCommand.Flags().BoolVar(&nfsMountEnabled, "nfs-mount-enabled", false, "enable NFS mounting of project in container")
	ConfigCommand.Flags().StringVar(&hostWebserverPortArg, "host-webserver-port", "", "The web container's localhost-bound port")
	ConfigCommand.Flags().StringVar(&hostHTTPSPortArg, "host-https-port", "", "The web container's localhost-bound https port")

	ConfigCommand.Flags().StringVar(&hostDBPortArg, "host-db-port", "", "The db container's localhost-bound port")
	ConfigCommand.Flags().StringVar(&phpMyAdminPortArg, "phpmyadmin-port", "", "Router port to be used for PHPMyAdmin (dba) container access")
	ConfigCommand.Flags().StringVar(&mailhogPortArg, "mailhog-port", "", "Router port to be used for mailhog access")

	// projectname flag exists for backwards compatability.
	ConfigCommand.Flags().StringVar(&projectNameArg, "projectname", "", projectNameUsage)
	err = ConfigCommand.Flags().MarkDeprecated("projectname", "The --projectname flag is deprecated in favor of --project-name")
	util.CheckErr(err)

	// apptype flag exists for backwards compatability.
	ConfigCommand.Flags().StringVar(&projectTypeArg, "projecttype", "", projectTypeUsage)
	err = ConfigCommand.Flags().MarkDeprecated("projecttype", "The --projecttype flag is deprecated in favor of --project-type")
	util.CheckErr(err)

	// apptype flag exists for backwards compatibility.
	ConfigCommand.Flags().StringVar(&projectTypeArg, "apptype", "", projectTypeUsage+" This is the same as --project-type and is included only for backwards compatibility.")
	err = ConfigCommand.Flags().MarkDeprecated("apptype", "The apptype flag is deprecated in favor of --project-type")
	util.CheckErr(err)

	// sitename flag exists for backwards compatibility.
	ConfigCommand.Flags().StringVar(&projectNameArg, "sitename", "", projectNameUsage+" This is the same as project-name and is included only for backwards compatibility")
	err = ConfigCommand.Flags().MarkDeprecated("sitename", "The sitename flag is deprecated in favor of --project-name")
	util.CheckErr(err)

	ConfigCommand.Flags().StringVar(&webimageExtraPackages, "webimage-extra-packages", "", "A comma-delimited list of Debian packages that should be added to web container when the project is started")

	ConfigCommand.Flags().StringVar(&dbimageExtraPackages, "dbimage-extra-packages", "", "A comma-delimited list of Debian packages that should be added to db container when the project is started")

	ConfigCommand.Flags().StringVar(&projectTLDArg, "project-tld", ddevapp.DdevDefaultTLD, "set the top-level domain to be used for projects, defaults to "+ddevapp.DdevDefaultTLD)

	ConfigCommand.Flags().BoolVarP(&useDNSWhenPossibleArg, "use-dns-when-possible", "", true, "Use DNS for hostname resolution instead of /etc/hosts when possible")

	RootCmd.AddCommand(ConfigCommand)
}

// getConfigApp() does the basic setup of the app (with provider) and returns it.
func getConfigApp(providerName string) (*ddevapp.DdevApp, error) {
	appRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not determine current working directory: %v", err)
	}

	// Check for an existing config in a parent dir
	otherRoot, _ := ddevapp.CheckForConf(appRoot)
	if otherRoot != "" && otherRoot != appRoot {
		util.Error("Is it possible you wanted to `ddev config` in parent directory %s?", otherRoot)
	}
	app, err := ddevapp.NewApp(appRoot, false, providerName)
	if err != nil {
		return nil, fmt.Errorf("could not create new config: %v", err)
	}
	return app, nil
}

// handleMainConfigArgs() validates and processes the main config args (docroot, etc.)
func handleMainConfigArgs(cmd *cobra.Command, args []string, app *ddevapp.DdevApp) error {
	var err error

	// Support the show-config-location flag.
	if showConfigLocation {
		activeApp, err := ddevapp.GetActiveApp("")
		if err != nil {
			if strings.Contains(err.Error(), "Have you run 'ddev config'") {
				util.Failed("No project configuration currently exists")
			} else {
				util.Failed("Failed to access project configuration: %v", err)
			}
		}

		if activeApp.ConfigPath != "" && activeApp.ConfigExists() {
			rawResult := make(map[string]interface{})
			rawResult["configpath"] = activeApp.ConfigPath
			rawResult["approot"] = activeApp.AppRoot

			friendlyMsg := fmt.Sprintf("The project config location is %s", activeApp.ConfigPath)
			output.UserOut.WithField("raw", rawResult).Print(friendlyMsg)
			return nil
		}
	}

	// Let them know if we're replacing the config.yaml
	app.WarnIfConfigReplace()

	// app.Name gets set to basename if not provided, or set to siteNameArg if provided
	if app.Name != "" && projectNameArg == "" { // If we already have a c.Name and no siteNameArg, leave c.Name alone
		// Sorry this is empty but it makes the logic clearer.
	} else if projectNameArg != "" { // if we have a siteNameArg passed in, use it for c.Name
		app.Name = projectNameArg
	} else { // No siteNameArg passed, c.Name not set: use c.Name from the directory
		pwd, err := os.Getwd()
		util.CheckErr(err)
		app.Name = filepath.Base(pwd)
	}

	// Ensure that the docroot exists
	if docrootRelPathArg != "" {
		app.Docroot = docrootRelPathArg
		if _, err = os.Stat(docrootRelPathArg); os.IsNotExist(err) {
			// If the user has indicated that the docroot should be created, create it.
			if !createDocroot {
				util.Failed("The provided docroot %s does not exist. Allow ddev to create it with the --create-docroot flag.", docrootRelPathArg)
			}

			var docrootAbsPath string
			docrootAbsPath, err = filepath.Abs(app.Docroot)
			if err != nil {
				util.Failed("Could not create docroot at %s: %v", docrootRelPathArg, err)
			}

			if err = os.MkdirAll(docrootAbsPath, 0755); err != nil {
				util.Failed("Could not create docroot at %s: %v", docrootAbsPath, err)
			}

			util.Success("Created docroot at %s", docrootAbsPath)
		}
	} else if !cmd.Flags().Changed("docroot") {
		app.Docroot = ddevapp.DiscoverDefaultDocroot(app)
	}

	if projectTypeArg != "" && !ddevapp.IsValidAppType(projectTypeArg) {
		validAppTypes := strings.Join(ddevapp.GetValidAppTypes(), ", ")
		util.Failed("apptype must be one of %s", validAppTypes)
	}

	detectedApptype := app.DetectAppType()
	fullPath, pathErr := filepath.Abs(app.Docroot)
	if pathErr != nil {
		util.Failed("Failed to get absolute path to Docroot %s: %v", app.Docroot, pathErr)
	}
	if projectTypeArg == "" || projectTypeArg == detectedApptype { // Found an app, matches passed-in or no apptype passed
		projectTypeArg = detectedApptype
		util.Success("Found a %s codebase at %s", detectedApptype, fullPath)
	} else if projectTypeArg != "" { // apptype was passed, but we found no app at all
		util.Warning("You have specified a project type of %s but no project of that type is found in %s", projectTypeArg, fullPath)
	} else if projectTypeArg != "" && detectedApptype != projectTypeArg { // apptype was passed, app was found, but not the same type
		util.Warning("You have specified a project type of %s but a project of type %s was discovered in %s", projectTypeArg, detectedApptype, fullPath)
	}
	app.Type = projectTypeArg

	// App overrides are done after app type is detected, but
	// before user-defined flags are set.
	err = app.ConfigFileOverrideAction()
	if err != nil {
		util.Failed("failed to run ConfigFileOverrideAction: %v", err)
	}

	// We don't want to write out dbimage if it's just the one that goes with
	// the mariadb_version.
	if app.DBImage == version.GetDBImage(app.MariaDBVersion) {
		app.DBImage = ""
	}

	if phpVersionArg != "" {
		app.PHPVersion = phpVersionArg
	}

	if httpPortArg != "" {
		app.RouterHTTPPort = httpPortArg
	}

	if httpsPortArg != "" {
		app.RouterHTTPSPort = httpsPortArg
	}

	if hostWebserverPortArg != "" {
		app.HostWebserverPort = hostWebserverPortArg
	}
	if hostHTTPSPortArg != "" {
		app.HostHTTPSPort = hostHTTPSPortArg
	}

	if hostDBPortArg != "" {
		app.HostDBPort = hostDBPortArg
	}

	// If the mariaDBVersionArg is set, use it
	if mariaDBVersionArg != "" {
		app.MariaDBVersion = mariaDBVersionArg
	}

	if cmd.Flag("nfs-mount-enabled").Changed {
		app.NFSMountEnabled = nfsMountEnabled
	}

	// This bool flag is false by default, so only use the value if the flag was explicity set.
	if cmd.Flag("xdebug-enabled").Changed {
		app.XdebugEnabled = xdebugEnabledArg
	}

	if cmd.Flag("phpmyadmin-port").Changed {
		app.PHPMyAdminPort = phpMyAdminPortArg
	}
	if cmd.Flag("mailhog-port").Changed {
		app.MailhogPort = mailhogPortArg
	}

	if additionalHostnamesArg != "" {
		app.AdditionalHostnames = strings.Split(additionalHostnamesArg, ",")
	}

	if additionalFQDNsArg != "" {
		app.AdditionalFQDNs = strings.Split(additionalFQDNsArg, ",")
	}

	if omitContainersArg != "" {
		app.OmitContainers = strings.Split(omitContainersArg, ",")
	}

	if cmd.Flag("webimage-extra-packages").Changed {
		if webimageExtraPackages == "" {
			app.WebImageExtraPackages = nil
		} else {
			app.WebImageExtraPackages = strings.Split(webimageExtraPackages, ",")
		}
	}

	if cmd.Flag("dbimage-extra-packages").Changed {
		if dbimageExtraPackages == "" {
			app.DBImageExtraPackages = nil
		} else {
			app.DBImageExtraPackages = strings.Split(dbimageExtraPackages, ",")
		}
	}

	if cmd.Flag("dbimage-extra-packages").Changed {
		app.WebImageExtraPackages = strings.Split(webimageExtraPackages, ",")
	}

	if cmd.Flag("use-dns-when-possible").Changed {
		app.UseDNSWhenPossible = useDNSWhenPossibleArg
	}

	if cmd.Flag("project-tld").Changed {
		app.ProjectTLD = projectTLDArg
	}

	if uploadDirArg != "" {
		app.UploadDir = uploadDirArg
	}

	if webserverTypeArg != "" {
		app.WebserverType = webserverTypeArg
	}

	if webImageArg != "" {
		app.WebImage = webImageArg
	}

	if webImageDefaultArg {
		app.WebImage = ""
	}

	if dbImageArg != "" {
		app.DBImage = dbImageArg
	}

	if dbImageDefaultArg {
		app.DBImage = ""
	}

	if dbaImageArg != "" {
		app.DBAImage = dbaImageArg
	}

	if dbaImageDefaultArg {
		app.DBAImage = ""
	}

	if imageDefaultsArg {
		app.WebImage = ""
		app.DBImage = ""
		app.DBAImage = ""
	}

	if app.WorkingDir == nil {
		app.WorkingDir = map[string]string{}
	}

	// Set working directory overrides
	if webWorkingDirArg != "" {
		app.WorkingDir["web"] = webWorkingDirArg
	}

	if dbWorkingDirArg != "" {
		app.WorkingDir["db"] = dbWorkingDirArg
	}

	if dbaWorkingDirArg != "" {
		app.WorkingDir["dba"] = dbaWorkingDirArg
	}

	// If default working directory overrides are requested, they take precedence
	defaults := app.DefaultWorkingDirMap()
	if workingDirDefaultsArg {
		app.WorkingDir = defaults
	}

	if webWorkingDirDefaultArg {
		app.WorkingDir["web"] = defaults["web"]
	}

	if dbWorkingDirDefaultArg {
		app.WorkingDir["db"] = defaults["db"]
	}

	if dbaWorkingDirDefaultArg {
		app.WorkingDir["dba"] = defaults["dba"]
	}

	// Ensure the configuration passes validation before writing config file.
	if err := app.ValidateConfig(); err != nil {
		return fmt.Errorf("failed to validate config: %v", err)
	}

	if err := app.WriteConfig(); err != nil {
		return fmt.Errorf("could not write ddev config file %s: %v", app.ConfigPath, err)
	}

	return nil
}
