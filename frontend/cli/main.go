package main

import (
	"context"
	"encoding/json"
	"fmt"
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

	// --- Motors ---
	motorClient := quadsmithconnect.NewMotorServiceClient(http.DefaultClient, apiURL)
	motorsCmd := &cobra.Command{Use: "motors"}
	motorsCmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			req := &pb.ListMotorsRequest{Filter: filter}
			res, err := motorClient.ListMotors(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg.Motors)
		},
	})
	motorsCmd.Commands()[0].Flags().StringP("filter", "f", "", "CEL filter string")

	motorsCmd.AddCommand(&cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &pb.GetMotorRequest{Id: args[0]}
			res, err := motorClient.GetMotor(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg)
		},
	})
	rootCmd.AddCommand(motorsCmd)

	// --- Frames ---
	frameClient := quadsmithconnect.NewFrameServiceClient(http.DefaultClient, apiURL)
	framesCmd := &cobra.Command{Use: "frames"}
	framesCmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			req := &pb.ListFramesRequest{Filter: filter}
			res, err := frameClient.ListFrames(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg.Frames)
		},
	})
	framesCmd.Commands()[0].Flags().StringP("filter", "f", "", "CEL filter string")

	framesCmd.AddCommand(&cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &pb.GetFrameRequest{Id: args[0]}
			res, err := frameClient.GetFrame(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg)
		},
	})
	rootCmd.AddCommand(framesCmd)

	// --- Batteries ---
	batteryClient := quadsmithconnect.NewBatteryServiceClient(http.DefaultClient, apiURL)
	batteriesCmd := &cobra.Command{Use: "batteries"}
	batteriesCmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			req := &pb.ListBatteriesRequest{Filter: filter}
			res, err := batteryClient.ListBatteries(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg.Batteries)
		},
	})
	batteriesCmd.Commands()[0].Flags().StringP("filter", "f", "", "CEL filter string")

	batteriesCmd.AddCommand(&cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &pb.GetBatteryRequest{Id: args[0]}
			res, err := batteryClient.GetBattery(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg)
		},
	})
	rootCmd.AddCommand(batteriesCmd)

	// --- Escs ---
	escClient := quadsmithconnect.NewEscServiceClient(http.DefaultClient, apiURL)
	escsCmd := &cobra.Command{Use: "escs"}
	escsCmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			req := &pb.ListEscsRequest{Filter: filter}
			res, err := escClient.ListEscs(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg.Escs)
		},
	})
	escsCmd.Commands()[0].Flags().StringP("filter", "f", "", "CEL filter string")

	escsCmd.AddCommand(&cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &pb.GetEscRequest{Id: args[0]}
			res, err := escClient.GetEsc(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg)
		},
	})
	rootCmd.AddCommand(escsCmd)

	// --- FlightControllers ---
	flightcontrollerClient := quadsmithconnect.NewFlightControllerServiceClient(http.DefaultClient, apiURL)
	flightcontrollersCmd := &cobra.Command{Use: "flightcontrollers"}
	flightcontrollersCmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			req := &pb.ListFlightControllersRequest{Filter: filter}
			res, err := flightcontrollerClient.ListFlightControllers(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg.FlightControllers)
		},
	})
	flightcontrollersCmd.Commands()[0].Flags().StringP("filter", "f", "", "CEL filter string")

	flightcontrollersCmd.AddCommand(&cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &pb.GetFlightControllerRequest{Id: args[0]}
			res, err := flightcontrollerClient.GetFlightController(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg)
		},
	})
	rootCmd.AddCommand(flightcontrollersCmd)

	// --- Receivers ---
	receiverClient := quadsmithconnect.NewReceiverServiceClient(http.DefaultClient, apiURL)
	receiversCmd := &cobra.Command{Use: "receivers"}
	receiversCmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			req := &pb.ListReceiversRequest{Filter: filter}
			res, err := receiverClient.ListReceivers(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg.Receivers)
		},
	})
	receiversCmd.Commands()[0].Flags().StringP("filter", "f", "", "CEL filter string")

	receiversCmd.AddCommand(&cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &pb.GetReceiverRequest{Id: args[0]}
			res, err := receiverClient.GetReceiver(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg)
		},
	})
	rootCmd.AddCommand(receiversCmd)

	// --- VideoTransmitters ---
	videotransmitterClient := quadsmithconnect.NewVideoTransmitterServiceClient(http.DefaultClient, apiURL)
	videotransmittersCmd := &cobra.Command{Use: "videotransmitters"}
	videotransmittersCmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			req := &pb.ListVideoTransmittersRequest{Filter: filter}
			res, err := videotransmitterClient.ListVideoTransmitters(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg.VideoTransmitters)
		},
	})
	videotransmittersCmd.Commands()[0].Flags().StringP("filter", "f", "", "CEL filter string")

	videotransmittersCmd.AddCommand(&cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &pb.GetVideoTransmitterRequest{Id: args[0]}
			res, err := videotransmitterClient.GetVideoTransmitter(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg)
		},
	})
	rootCmd.AddCommand(videotransmittersCmd)

	// --- Antennas ---
	antennaClient := quadsmithconnect.NewAntennaServiceClient(http.DefaultClient, apiURL)
	antennasCmd := &cobra.Command{Use: "antennas"}
	antennasCmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			req := &pb.ListAntennasRequest{Filter: filter}
			res, err := antennaClient.ListAntennas(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg.Antennas)
		},
	})
	antennasCmd.Commands()[0].Flags().StringP("filter", "f", "", "CEL filter string")

	antennasCmd.AddCommand(&cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &pb.GetAntennaRequest{Id: args[0]}
			res, err := antennaClient.GetAntenna(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg)
		},
	})
	rootCmd.AddCommand(antennasCmd)

	// --- Cameras ---
	cameraClient := quadsmithconnect.NewCameraServiceClient(http.DefaultClient, apiURL)
	camerasCmd := &cobra.Command{Use: "cameras"}
	camerasCmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			req := &pb.ListCamerasRequest{Filter: filter}
			res, err := cameraClient.ListCameras(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg.Cameras)
		},
	})
	camerasCmd.Commands()[0].Flags().StringP("filter", "f", "", "CEL filter string")

	camerasCmd.AddCommand(&cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &pb.GetCameraRequest{Id: args[0]}
			res, err := cameraClient.GetCamera(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg)
		},
	})
	rootCmd.AddCommand(camerasCmd)

	// --- Propellers ---
	propellerClient := quadsmithconnect.NewPropellerServiceClient(http.DefaultClient, apiURL)
	propellersCmd := &cobra.Command{Use: "propellers"}
	propellersCmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			req := &pb.ListPropellersRequest{Filter: filter}
			res, err := propellerClient.ListPropellers(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg.Propellers)
		},
	})
	propellersCmd.Commands()[0].Flags().StringP("filter", "f", "", "CEL filter string")

	propellersCmd.AddCommand(&cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &pb.GetPropellerRequest{Id: args[0]}
			res, err := propellerClient.GetPropeller(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg)
		},
	})
	rootCmd.AddCommand(propellersCmd)

	// --- Builds ---
	buildClient := quadsmithconnect.NewBuildServiceClient(http.DefaultClient, apiURL)
	buildsCmd := &cobra.Command{Use: "builds"}
	buildsCmd.AddCommand(&cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			req := &pb.ListBuildsRequest{Filter: filter}
			res, err := buildClient.ListBuilds(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg.Builds)
		},
	})
	buildsCmd.Commands()[0].Flags().StringP("filter", "f", "", "CEL filter string")

	buildsCmd.AddCommand(&cobra.Command{
		Use:  "get [id]",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &pb.GetBuildRequest{Id: args[0]}
			res, err := buildClient.GetBuild(context.Background(), connect.NewRequest(req))
			if err != nil {
				return err
			}
			return printOutput(res.Msg)
		},
	})
	rootCmd.AddCommand(buildsCmd)

	// --- EVALUATOR ---
	evalClient := quadsmithconnect.NewEvaluatorServiceClient(http.DefaultClient, apiURL)
	evalCmd := &cobra.Command{
		Use:   "evaluate [build-id]",
		Short: "Run physics estimation and compatibility checks on a build",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bReq := &pb.GetBuildRequest{Id: args[0]}
			bRes, err := buildClient.GetBuild(context.Background(), connect.NewRequest(bReq))
			if err != nil {
				return fmt.Errorf("failed to fetch build: %w", err)
			}

			payload, _ := cmd.Flags().GetFloat32("payload")
			eReq := &pb.EvaluateBuildRequest{
				Build:          bRes.Msg,
				PayloadWeightG: payload,
			}
			eRes, err := evalClient.EvaluateBuild(context.Background(), connect.NewRequest(eReq))
			if err != nil {
				return fmt.Errorf("evaluation failed: %w", err)
			}

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
