package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func createResourceCmd[T proto.Message](
	useName string,
	cType string,
	defaultCols []string,
	listFunc func(ctx context.Context, filter string) (proto.Message, []T, error),
	getFunc func(ctx context.Context, id string) (proto.Message, error),
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   useName,
		Short: fmt.Sprintf("Manage %s", useName),
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("List all %s", useName),
		Run: func(c *cobra.Command, args []string) {
			filterStr, _ := c.Flags().GetString("filter")
			resMsg, items, err := listFunc(context.Background(), filterStr)
			if err != nil {
				log.Fatalf("Search failed: %v", err)
			}
			if jsonOut || yamlOut {
				fmt.Println(formatOutput(resMsg))
				return
			}
			cols := defaultCols
			if columns != "" {
				cols = strings.Split(columns, ",")
			}
			printTable(items, resolveCols(cols, cType))
		},
	}
	listCmd.Flags().String("filter", "", "CEL filter string (e.g. 'motor.kv_rating > 2000')")

	describeCmd := &cobra.Command{
		Use:   "describe [id]",
		Short: fmt.Sprintf("Describe a specific %s", useName),
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(c *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			getClient()
			_, res, err := listFunc(context.Background(), "")
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			var completions []string

			for _, item := range res {
				m := protoToMap(item)
				if idVal, ok := m["id"]; ok {
					if idStr, ok := idVal.(string); ok {
						if strings.HasPrefix(idStr, toComplete) {
							completions = append(completions, idStr)
						}
					}
				}
			}

			return completions, cobra.ShellCompDirectiveNoFileComp
		},
		Run: func(c *cobra.Command, args []string) {
			id := args[0]
			resMsg, err := getFunc(context.Background(), id)
			if err != nil {
				log.Fatalf("Failed to get %s: %v", useName, err)
			}
			fmt.Println(formatOutput(resMsg))
		},
	}

	cmd.AddCommand(listCmd, describeCmd)
	return cmd
}
