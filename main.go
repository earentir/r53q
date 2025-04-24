package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/spf13/cobra"
)

var (
	// injected via ldflags
	appVersion = "dev"
	gitCommit  = "none"
	buildDate  = "unknown"

	showVersion bool
)

// config holds AWS creds & region
type config struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Region    string `json:"region"`
}

// loadConfigAndSource locates or creates a config, or loads from env.
// Returns (*config, source, path, error)
// source is "file", "env", or "created"
func loadConfigAndSource() (*config, string, string, error) {
	// 1) next to binary
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		p := filepath.Join(dir, "r53q.json")
		if _, err := os.Stat(p); err == nil {
			// load file
			cfg, err := loadconfig(p)
			return cfg, "file", p, err
		}
	}
	// 2) $HOME/.config
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, ".config", "r53q.json")
		if _, err := os.Stat(p); err == nil {
			cfg, err := loadconfig(p)
			return cfg, "file", p, err
		}
	}
	// 3) /etc/r53q.json
	etcp := "/etc/r53q.json"
	if _, err := os.Stat(etcp); err == nil {
		cfg, err := loadconfig(etcp)
		return cfg, "file", etcp, err
	}
	// 4) env vars
	access := os.Getenv("AWS_ACCESS_KEY_ID")
	secret := os.Getenv("AWS_SECRET_ACCESS_KEY")
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}
	if access != "" && secret != "" && region != "" {
		return &config{access, secret, region}, "env", "", nil
	}
	// 5) none: create empty in cwd
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", "", err
	}
	p := filepath.Join(cwd, "r53q.json")
	empty := &config{}
	data, _ := json.MarshalIndent(empty, "", "  ")
	if err := os.WriteFile(p, data, 0644); err != nil {
		return nil, "", "", err
	}
	return empty, "created", p, nil
}

// loadconfig reads AWS creds & region from JSON file
func loadconfig(path string) (*config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// listZones prints a nice table of all hosted zones
func listZones(cfg *config) error {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(cfg.Region),
		Credentials: credentials.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey, ""),
	})
	if err != nil {
		return err
	}
	svc := route53.New(sess)

	rows := [][]string{{"ID", "Name", "Records"}}
	if err := svc.ListHostedZonesPages(&route53.ListHostedZonesInput{},
		func(out *route53.ListHostedZonesOutput, last bool) bool {
			for _, z := range out.HostedZones {
				rows = append(rows, []string{
					strings.TrimPrefix(aws.StringValue(z.Id), "/hostedzone/"),
					aws.StringValue(z.Name),
					fmt.Sprintf("%d", aws.Int64Value(z.ResourceRecordSetCount)),
				})
			}
			return !last
		}); err != nil {
		return err
	}

	// align columns
	widths := make([]int, len(rows[0]))
	for _, r := range rows {
		for i, c := range r {
			if len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
	}
	for ri, r := range rows {
		for i, c := range r {
			cell := c
			if ri == 0 {
				cell = strings.ToUpper(c)
			}
			fmt.Printf("%-*s  ", widths[i], cell)
		}
		fmt.Println()
	}
	return nil
}

// listRecords prints all records in a zone (by ID or domain)
func listRecords(cfg *config, identifier string) error {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(cfg.Region),
		Credentials: credentials.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey, ""),
	})
	if err != nil {
		return err
	}
	svc := route53.New(sess)

	dom := identifier
	isDomain := strings.Contains(identifier, ".")
	if isDomain && !strings.HasSuffix(dom, ".") {
		dom += "."
	}

	// resolve zone ID
	outZones, err := svc.ListHostedZones(&route53.ListHostedZonesInput{})
	if err != nil {
		return err
	}
	var zoneID string
	for _, z := range outZones.HostedZones {
		idVal := aws.StringValue(z.Id)
		nameVal := aws.StringValue(z.Name)
		if (isDomain && nameVal == dom) ||
			(!isDomain && (idVal == identifier || idVal == "/hostedzone/"+identifier)) {
			zoneID = idVal
			break
		}
	}
	if zoneID == "" {
		return fmt.Errorf("no hosted zone found for %q", identifier)
	}

	// collect records
	rows := [][]string{{"Name", "Type", "TTL", "Values"}}
	if err := svc.ListResourceRecordSetsPages(&route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
	}, func(out *route53.ListResourceRecordSetsOutput, last bool) bool {
		for _, rr := range out.ResourceRecordSets {
			vals := make([]string, len(rr.ResourceRecords))
			for i, r := range rr.ResourceRecords {
				vals[i] = aws.StringValue(r.Value)
			}
			rows = append(rows, []string{
				aws.StringValue(rr.Name),
				aws.StringValue(rr.Type),
				fmt.Sprintf("%d", aws.Int64Value(rr.TTL)),
				strings.Join(vals, ", "),
			})
		}
		return !last
	}); err != nil {
		return err
	}

	// align & print
	widths := make([]int, len(rows[0]))
	for _, r := range rows {
		for i, c := range r {
			if len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
	}
	for ri, r := range rows {
		for i, c := range r {
			cell := c
			if ri == 0 {
				cell = strings.ToUpper(c)
			}
			fmt.Printf("%-*s  ", widths[i], cell)
		}
		fmt.Println()
	}
	return nil
}

// zoneInfo prints either the ID/name or count for one zone
func zoneInfo(cfg *config, identifier string, countOnly bool) error {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(cfg.Region),
		Credentials: credentials.NewStaticCredentials(cfg.AccessKey, cfg.SecretKey, ""),
	})
	if err != nil {
		return err
	}
	svc := route53.New(sess)

	dom := identifier
	isDomain := strings.Contains(identifier, ".")
	if isDomain && !strings.HasSuffix(dom, ".") {
		dom += "."
	}

	outZones, err := svc.ListHostedZones(&route53.ListHostedZonesInput{})
	if err != nil {
		return err
	}
	var foundID, foundName string
	var recordCount int64
	for _, z := range outZones.HostedZones {
		idVal := aws.StringValue(z.Id)
		nameVal := aws.StringValue(z.Name)
		if (isDomain && nameVal == dom) ||
			(!isDomain && (idVal == identifier || idVal == "/hostedzone/"+identifier)) {
			foundID = idVal
			foundName = nameVal
			recordCount = aws.Int64Value(z.ResourceRecordSetCount)
			break
		}
	}
	if foundID == "" {
		return fmt.Errorf("no hosted zone found for %q", identifier)
	}

	if countOnly {
		fmt.Println(recordCount)
	} else if isDomain {
		fmt.Println(strings.TrimPrefix(foundID, "/hostedzone/"))
	} else {
		fmt.Println(strings.TrimSuffix(foundName, "."))
	}
	return nil
}

func main() {
	root := &cobra.Command{
		Use:   "r53q",
		Short: "Tiny Route53 CLI",
		Run: func(cmd *cobra.Command, args []string) {
			if showVersion {
				// print version
				fmt.Printf("r53q %s (commit %s, built %s)\n",
					appVersion, gitCommit, buildDate)
				// show config source
				_, src, path, _ := loadConfigAndSource()
				switch src {
				case "file":
					fmt.Printf("Config: %s\n", path)
				case "env":
					fmt.Println("Config: environment")
				case "created":
					fmt.Printf("Config: created at %s (please fill in credentials)\n", path)
				}
				os.Exit(0)
			}
			cmd.Help()
		},
	}

	// global version flag
	root.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version & config path, then exit")

	// list/zones
	list := &cobra.Command{Use: "list", Short: "List Route53 resources"}
	zones := &cobra.Command{
		Use:   "zones",
		Short: "List hosted Route53 zones",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, src, path, err := loadConfigAndSource()
			if err != nil {
				log.Fatalf("config error: %v", err)
			}
			if src == "created" {
				log.Fatalf("No config found; created %s with empty values. Please populate credentials.", path)
			}
			if err := listZones(cfg); err != nil {
				log.Fatalf("list zones failed: %v", err)
			}
		},
	}
	list.AddCommand(zones)

	// list records
	records := &cobra.Command{
		Use:   "records <zone-id|domain>",
		Short: "List all records in a hosted zone",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg, src, path, err := loadConfigAndSource()
			if err != nil {
				log.Fatalf("config error: %v", err)
			}
			if src == "created" {
				log.Fatalf("No config found; created %s with empty values. Please populate credentials.", path)
			}
			if err := listRecords(cfg, args[0]); err != nil {
				log.Fatalf("list records failed: %v", err)
			}
		},
	}
	list.AddCommand(records)

	// zone info
	zone := &cobra.Command{
		Use:   "zone <zone-id|domain> [count]",
		Short: "Return a zone’s ID/name (default) or record count",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			cfg, src, path, err := loadConfigAndSource()
			if err != nil {
				log.Fatalf("config error: %v", err)
			}
			if src == "created" {
				log.Fatalf("No config found; created %s with empty values. Please populate credentials.", path)
			}
			countOnly := len(args) == 2 && strings.ToLower(args[1]) == "count"
			if err := zoneInfo(cfg, args[0], countOnly); err != nil {
				log.Fatalf("zone info failed: %v", err)
			}
		},
	}

	root.AddCommand(list, zone)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
