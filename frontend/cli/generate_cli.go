// +build ignore

package main

import (
	"fmt"
	"os"
	"strings"
)

var domains = []struct {
	Name   string
	Plural string
}{
	{"Motor", "Motors"},
	{"Frame", "Frames"},
	{"Battery", "Batteries"},
	{"Esc", "Escs"},
	{"FlightController", "FlightControllers"},
	{"Receiver", "Receivers"},
	{"VideoTransmitter", "VideoTransmitters"},
	{"Antenna", "Antennas"},
	{"Camera", "Cameras"},
	{"Propeller", "Propellers"},
	{"Build", "Builds"},
}

func main() {
	f, _ := os.Create("main.go")
	defer f.Close()

	fmt.Fprintln(f, `package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
)

var (
	apiURL = "http://localhost:8080"
	output = "yaml"
)

func main() {
	if url := os.Getenv("QS_API_URL"); url != "" {
		apiURL = url
	}

	rootCmd := &cobra.Command{
		Use:   "qs",
		Short: "Quadsmith CLI",
	}
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "yaml", "Output format (yaml|json)")
`)

	for _, d := range domains {
		lowerName := strings.ToLower(d.Name)
		lowerPlural := strings.ToLower(d.Plural)
		
		fmt.Fprintf(f, "\n\t// --- %s ---\n", d.Plural)
		fmt.Fprintf(f, "\t%sClient := quadsmithconnect.New%sServiceClient(http.DefaultClient, apiURL)\n", lowerName, d.Name)
		fmt.Fprintf(f, "\t%sCmd := &cobra.Command{Use: \"%s\"}\n", lowerPlural, lowerPlural)
		
		fmt.Fprintf(f, "\t%sCmd.AddCommand(&cobra.Command{\n", lowerPlural)
		fmt.Fprintf(f, "\t\tUse: \"list\",\n")
		fmt.Fprintf(f, "\t\tRunE: func(cmd *cobra.Command, args []string) error {\n")
		fmt.Fprintf(f, "\t\t\tfilter, _ := cmd.Flags().GetString(\"filter\")\n")
		fmt.Fprintf(f, "\t\t\treq := &pb.List%sRequest{Filter: filter}\n", d.Plural)
		fmt.Fprintf(f, "\t\t\tres, err := %sClient.List%s(context.Background(), connect.NewRequest(req))\n", lowerName, d.Plural)
		fmt.Fprintf(f, "\t\t\tif err != nil { return err }\n")
		fmt.Fprintf(f, "\t\t\treturn printOutput(res.Msg.%s)\n", d.Plural)
		fmt.Fprintf(f, "\t\t},\n")
		fmt.Fprintf(f, "\t})\n")
		fmt.Fprintf(f, "\t%sCmd.Commands()[0].Flags().StringP(\"filter\", \"f\", \"\", \"CEL filter string\")\n\n", lowerPlural)

		fmt.Fprintf(f, "\t%sCmd.AddCommand(&cobra.Command{\n", lowerPlural)
		fmt.Fprintf(f, "\t\tUse: \"get [id]\",\n")
		fmt.Fprintf(f, "\t\tArgs: cobra.ExactArgs(1),\n")
		fmt.Fprintf(f, "\t\tValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {\n")
		fmt.Fprintf(f, "\t\t\tif len(args) != 0 { return nil, cobra.ShellCompDirectiveNoFileComp }\n")
		fmt.Fprintf(f, "\t\t\tres, err := %sClient.List%s(context.Background(), connect.NewRequest(&pb.List%sRequest{}))\n", lowerName, d.Plural, d.Plural)
		fmt.Fprintf(f, "\t\t\tif err != nil { return nil, cobra.ShellCompDirectiveError }\n")
		fmt.Fprintf(f, "\t\t\tvar comps []string\n")
		fmt.Fprintf(f, "\t\t\tfor _, item := range res.Msg.%s {\n", d.Plural)
		fmt.Fprintf(f, "\t\t\t\tcomps = append(comps, item.Id)\n") // Assuming every domain has an 'Id' field
		fmt.Fprintf(f, "\t\t\t}\n")
		fmt.Fprintf(f, "\t\t\treturn comps, cobra.ShellCompDirectiveNoFileComp\n")
		fmt.Fprintf(f, "\t\t},\n")
		fmt.Fprintf(f, "\t\tRunE: func(cmd *cobra.Command, args []string) error {\n")
		fmt.Fprintf(f, "\t\t\treq := &pb.Get%sRequest{Id: args[0]}\n", d.Name)
		fmt.Fprintf(f, "\t\t\tres, err := %sClient.Get%s(context.Background(), connect.NewRequest(req))\n", lowerName, d.Name)
		fmt.Fprintf(f, "\t\t\tif err != nil { return err }\n")
		fmt.Fprintf(f, "\t\t\treturn printOutput(res.Msg)\n")
		fmt.Fprintf(f, "\t\t},\n")
		fmt.Fprintf(f, "\t})\n")

		fmt.Fprintf(f, "\trootCmd.AddCommand(%sCmd)\n", lowerPlural)
	}

	fmt.Fprintln(f, `
	// --- EVALUATOR ---
	evalClient := quadsmithconnect.NewEvaluatorServiceClient(http.DefaultClient, apiURL)
	evalCmd := &cobra.Command{
		Use: "evaluate [build-id]",
		Short: "Run physics estimation and compatibility checks on a build",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 { return nil, cobra.ShellCompDirectiveNoFileComp }
			res, err := buildClient.ListBuilds(context.Background(), connect.NewRequest(&pb.ListBuildsRequest{}))
			if err != nil { return nil, cobra.ShellCompDirectiveError }
			var comps []string
			for _, item := range res.Msg.Builds {
				comps = append(comps, item.Id)
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			bReq := &pb.GetBuildRequest{Id: args[0]}
			bRes, err := buildClient.GetBuild(context.Background(), connect.NewRequest(bReq))
			if err != nil { return fmt.Errorf("failed to fetch build: %w", err) }
			
			payload, _ := cmd.Flags().GetFloat32("payload")
			eReq := &pb.EvaluateBuildRequest{
				Build: bRes.Msg,
				PayloadWeightG: payload,
			}
			eRes, err := evalClient.EvaluateBuild(context.Background(), connect.NewRequest(eReq))
			if err != nil { return fmt.Errorf("evaluation failed: %w", err) }
			
			return printOutput(eRes.Msg)
		},
	}
	evalCmd.Flags().Float32("payload", 0, "Payload weight in grams")
	rootCmd.AddCommand(evalCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func printOutput(data interface{}) error {
	var b []byte
	var err error
	if output == "json" {
		b, err = json.MarshalIndent(data, "", "  ")
	} else {
		b, err = yaml.Marshal(data)
	}
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
`)
}
