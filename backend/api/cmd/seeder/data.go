package main

import (
	"context"
	"github.com/jackc/pgx/v5"
	pb "quadsmith/api/gen/quadsmith"
)

func seedData(ctx context.Context, tx pgx.Tx) {

	if err := pb.CreateVideoTransmitter(ctx, tx, &pb.VideoTransmitter{
		Uuid: "91c711fe-590b-4aa5-b8c8-dd94787ee71f", Id: "dji-o3-air-unit", Manufacturer: "DJI", Model: "DJI O3 Air Unit",
		WeightG: 36.4, Protocol: "Digital", MaxPowerMw: 1995,
	}); err != nil { panic(err) }
	if err := pb.CreateEsc(ctx, tx, &pb.Esc{
		Uuid: "59b0e835-da96-4143-9674-dc6a950ac7c7", Id: "redfox-a3-f722-45a-aio", Manufacturer: "Redfox", Model: "Redfox A3 F722 45A AIO",
		WeightG: 8.7, ContinuousAmps: 45, BurstAmps: 55, MaxMotors: 4, Firmware: "BLHeli_32",
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "717058a0-4631-4b7e-9648-21273ceefb73", Id: "sub250-1404-4500kv", Manufacturer: "Sub250", Model: "Sub250 1404 4500KV",
		WeightG: 10.3, StatorDiameterMm: 14.0, StatorHeightMm: 4.0, Kv: 4500,
	}); err != nil { panic(err) }
	if err := pb.CreateFrame(ctx, tx, &pb.Frame{
		Uuid: "4315b6cb-1157-46ff-afd5-d16c06e30e45", Id: "tbs-source-one-v5", Manufacturer: "TBS", Model: "TBS Source One V5",
		WeightG: 123.5, WheelbaseMm: 226.0, MaxPropSizeInches: 5.1,
		Geometry: "True-X",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "5bb5b8ae-fc45-45c0-a388-11e0d0ab6124", Id: "hqprop-5x4.3x3-v1s", Manufacturer: "HQProp", Model: "HQProp 5x4.3x3 V1S",
		WeightG: 3.81, DiameterInches: 5.0, PitchInches: 4.3, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "1f582787-4b5d-4b39-b923-962988c15a5a", Id: "tattu-r-line-version-5.0-1200mah-6s", Manufacturer: "Tattu", Model: "Tattu R-Line Version 5.0 1200mAh 6S",
		WeightG: 197.0, CapacityMah: 1200, CellCountS: 6,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateCamera(ctx, tx, &pb.Camera{
		Uuid: "78744510-5235-4797-a8bd-dd85ee30d326", Id: "runcam-phoenix-2", Manufacturer: "RunCam", Model: "RunCam Phoenix 2",
		WeightG: 9.0, Protocol: "Analog", WidthMm: 19,
	}); err != nil { panic(err) }
	if err := pb.CreateReceiver(ctx, tx, &pb.Receiver{
		Uuid: "b0cf2152-68ff-41ac-9bc5-c7b6d34f2543", Id: "tbs-crossfire-nano-rx", Manufacturer: "TBS", Model: "TBS Crossfire Nano RX",
		WeightG: 0.5, Protocol: "TBS Crossfire (CRSF)", FrequencyBandGhz: 2.4,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "ad7b06d2-9f45-4f75-9e16-bf97d420dd9b", Id: "lumenier-axii-2-(rhcp)", Manufacturer: "Lumenier", Model: "Lumenier AXII 2 (RHCP)",
		WeightG: 4.8, Connector: "SMA", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateVideoTransmitter(ctx, tx, &pb.VideoTransmitter{
		Uuid: "b02d6857-65a7-4f44-a0cf-03b3ad33d791", Id: "caddx-vista", Manufacturer: "Caddx", Model: "Caddx Vista",
		WeightG: 19.0, Protocol: "Digital", MaxPowerMw: 1000,
	}); err != nil { panic(err) }
	if err := pb.CreateCamera(ctx, tx, &pb.Camera{
		Uuid: "a4c24db1-9024-4e23-9953-3dc8b9613673", Id: "walksnail-avatar-hd-pro-camera", Manufacturer: "Walksnail", Model: "Walksnail Avatar HD Pro Camera",
		WeightG: 9.5, Protocol: "Digital", WidthMm: 19,
	}); err != nil { panic(err) }
	if err := pb.CreateVideoTransmitter(ctx, tx, &pb.VideoTransmitter{
		Uuid: "57dea1f1-ee90-4e3d-a286-f7e88727d5bc", Id: "walksnail-avatar-vtx-v2", Manufacturer: "Walksnail", Model: "Walksnail Avatar VTX V2",
		WeightG: 15.4, Protocol: "Digital", MaxPowerMw: 1200,
	}); err != nil { panic(err) }
	if err := pb.CreateFrame(ctx, tx, &pb.Frame{
		Uuid: "73382232-1424-4e72-aed7-6a663a5608ec", Id: "impulserc-apex-evo-5\"", Manufacturer: "ImpulseRC", Model: "ImpulseRC Apex EVO 5\"",
		WeightG: 132.0, WheelbaseMm: 225.0, MaxPropSizeInches: 5.1,
		Geometry: "True-X",
	}); err != nil { panic(err) }
	if err := pb.CreateFrame(ctx, tx, &pb.Frame{
		Uuid: "295b1a5b-c9c9-49bd-9eb3-93422d525bf8", Id: "iflight-aos-5-v5", Manufacturer: "iFlight", Model: "iFlight AOS 5 V5",
		WeightG: 129.0, WheelbaseMm: 228.0, MaxPropSizeInches: 5.2,
		Geometry: "True-X",
	}); err != nil { panic(err) }
	if err := pb.CreateFrame(ctx, tx, &pb.Frame{
		Uuid: "498c4fd2-cf55-4022-ac48-cc400e490313", Id: "flyfishrc-volador-ii-vx5", Manufacturer: "FlyFishRC", Model: "FlyFishRC Volador II VX5",
		WeightG: 177.0, WheelbaseMm: 225.0, MaxPropSizeInches: 5.1,
		Geometry: "True-X",
	}); err != nil { panic(err) }
	if err := pb.CreateCamera(ctx, tx, &pb.Camera{
		Uuid: "1555417e-ba1b-4eb5-ae46-810b0b7b7900", Id: "caddx-nebula-pro", Manufacturer: "Caddx", Model: "Caddx Nebula Pro",
		WeightG: 6.0, Protocol: "Digital", WidthMm: 19,
	}); err != nil { panic(err) }
	if err := pb.CreateReceiver(ctx, tx, &pb.Receiver{
		Uuid: "8e3b3fae-9076-400d-89c3-f93faf3aa37d", Id: "walksnail-avatar-vrx", Manufacturer: "Walksnail", Model: "Walksnail Avatar VRX",
		WeightG: 83.0, Protocol: "HDMI", FrequencyBandGhz: 2.4,
	}); err != nil { panic(err) }
	if err := pb.CreateFrame(ctx, tx, &pb.Frame{
		Uuid: "8fd51fba-8f88-4178-9797-f73ad4e1b45b", Id: "caddx-gofilm-20-frame", Manufacturer: "Caddx", Model: "Caddx Gofilm 20 Frame",
		WeightG: 30.5, WheelbaseMm: 94.0, MaxPropSizeInches: 2.01,
		Geometry: "True-X",
	}); err != nil { panic(err) }
	if err := pb.CreateEsc(ctx, tx, &pb.Esc{
		Uuid: "5a5be012-c0c3-4986-99dd-97a7317148ce", Id: "caddxf4-aio-20a", Manufacturer: "CaddxF4", Model: "CaddxF4 AIO 20A",
		WeightG: 5.3, ContinuousAmps: 20, BurstAmps: 25, MaxMotors: 4, Firmware: "BLHeli_32",
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "db2dcc3e-efcc-42bb-89a6-9006d260729d", Id: "caddx-1303-6000kv", Manufacturer: "Caddx", Model: "Caddx 1303 6000KV",
		WeightG: 5.6, StatorDiameterMm: 13.0, StatorHeightMm: 3.0, Kv: 6000,
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "85e4b6d3-aeed-45c2-b51b-c55359528c7c", Id: "hqprop-t2x2x3", Manufacturer: "HQProp", Model: "HQProp T2x2x3",
		WeightG: 1.5, DiameterInches: 2.0, PitchInches: 2.0, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreateCamera(ctx, tx, &pb.Camera{
		Uuid: "730c445e-fbf3-4738-910e-3a5aadfa9434", Id: "walksnail-moonlight-camera", Manufacturer: "Walksnail", Model: "Walksnail Moonlight Camera",
		WeightG: 8.9, Protocol: "Digital", WidthMm: 19,
	}); err != nil { panic(err) }
	if err := pb.CreateVideoTransmitter(ctx, tx, &pb.VideoTransmitter{
		Uuid: "1c15772d-b52e-4871-aba6-12132995753f", Id: "walksnail-moonlight-vtx", Manufacturer: "Walksnail", Model: "Walksnail Moonlight VTX",
		WeightG: 38.5, Protocol: "Digital", MaxPowerMw: 1000,
	}); err != nil { panic(err) }
	if err := pb.CreateReceiver(ctx, tx, &pb.Receiver{
		Uuid: "632437ac-1938-4574-85eb-b9b2aeb000ff", Id: "tbs-tracer-nano-rx", Manufacturer: "TBS", Model: "TBS Tracer Nano RX",
		WeightG: 0.5, Protocol: "TBS Crossfire (CRSF)", FrequencyBandGhz: 2.4,
	}); err != nil { panic(err) }
	if err := pb.CreateVideoTransmitter(ctx, tx, &pb.VideoTransmitter{
		Uuid: "ecc4d1d6-b8da-4f13-bd14-737c5eb17b08", Id: "tbs-unify-pro32-nano", Manufacturer: "TBS", Model: "TBS Unify Pro32 Nano",
		WeightG: 1.0, Protocol: "", MaxPowerMw: 500,
	}); err != nil { panic(err) }
	if err := pb.CreateVideoTransmitter(ctx, tx, &pb.VideoTransmitter{
		Uuid: "6cc9ebbb-069f-4cef-bba7-cb93a3f98e09", Id: "walksnail-avatar-hd-mini-1s-vtx", Manufacturer: "Walksnail", Model: "Walksnail Avatar HD Mini 1S VTX",
		WeightG: 8.7, Protocol: "Digital", MaxPowerMw: 350,
	}); err != nil { panic(err) }
	if err := pb.CreateCamera(ctx, tx, &pb.Camera{
		Uuid: "a55cc596-b860-49a4-b08d-7c7d448d08ea", Id: "walksnail-avatar-hd-nano-camera-v3", Manufacturer: "Walksnail", Model: "Walksnail Avatar HD Nano Camera V3",
		WeightG: 3.0, Protocol: "Digital", WidthMm: 14,
	}); err != nil { panic(err) }
	if err := pb.CreateVideoTransmitter(ctx, tx, &pb.VideoTransmitter{
		Uuid: "77300a2a-bedb-41eb-9212-c01ce36fdff1", Id: "hdzero-freestyle-v2-vtx", Manufacturer: "HDZero", Model: "HDZero Freestyle V2 VTX",
		WeightG: 22.3, Protocol: "Digital", MaxPowerMw: 1000,
	}); err != nil { panic(err) }
	if err := pb.CreateVideoTransmitter(ctx, tx, &pb.VideoTransmitter{
		Uuid: "4d2cee14-b504-40b2-bfc7-92bb33c6d149", Id: "hdzero-whoop-lite-vtx", Manufacturer: "HDZero", Model: "HDZero Whoop Lite VTX",
		WeightG: 4.5, Protocol: "Digital", MaxPowerMw: 200,
	}); err != nil { panic(err) }
	if err := pb.CreateCamera(ctx, tx, &pb.Camera{
		Uuid: "956b292e-8f19-4434-9fc0-dccfdf7cec6e", Id: "hdzero-nano-90-camera", Manufacturer: "HDZero", Model: "HDZero Nano 90 Camera",
		WeightG: 5.2, Protocol: "Digital", WidthMm: 14,
	}); err != nil { panic(err) }
	if err := pb.CreateVideoTransmitter(ctx, tx, &pb.VideoTransmitter{
		Uuid: "59135644-10cc-4210-87b7-8c5a98970a8d", Id: "dji-o4-lite-vtx", Manufacturer: "DJI", Model: "DJI O4 Lite VTX",
		WeightG: 5.0, Protocol: "Digital", MaxPowerMw: 1000,
	}); err != nil { panic(err) }
	if err := pb.CreateCamera(ctx, tx, &pb.Camera{
		Uuid: "37198b31-b5bd-4c9b-918a-ef9ae5e0f5f8", Id: "dji-o4-lite-camera", Manufacturer: "DJI", Model: "DJI O4 Lite Camera",
		WeightG: 3.2, Protocol: "Digital", WidthMm: 14,
	}); err != nil { panic(err) }
	if err := pb.CreateVideoTransmitter(ctx, tx, &pb.VideoTransmitter{
		Uuid: "dc164c44-e652-40b9-aec9-000ff26cd26a", Id: "dji-o4-wide-vtx", Manufacturer: "DJI", Model: "DJI O4 Wide VTX",
		WeightG: 5.1, Protocol: "Digital", MaxPowerMw: 2000,
	}); err != nil { panic(err) }
	if err := pb.CreateCamera(ctx, tx, &pb.Camera{
		Uuid: "cc25d9bf-5c8c-47d0-820f-59effadeb75a", Id: "dji-o4-wide-camera", Manufacturer: "DJI", Model: "DJI O4 Wide Camera",
		WeightG: 8.3, Protocol: "Digital", WidthMm: 20,
	}); err != nil { panic(err) }
	if err := pb.CreateVideoTransmitter(ctx, tx, &pb.VideoTransmitter{
		Uuid: "76d2c270-7162-4878-be8d-ad2296223ba0", Id: "dji-o4-pro-vtx", Manufacturer: "DJI", Model: "DJI O4 Pro VTX",
		WeightG: 15.6, Protocol: "Digital", MaxPowerMw: 2000,
	}); err != nil { panic(err) }
	if err := pb.CreateCamera(ctx, tx, &pb.Camera{
		Uuid: "b09458dc-419a-4be7-9585-f1e40f8bfb7a", Id: "dji-o4-pro-camera", Manufacturer: "DJI", Model: "DJI O4 Pro Camera",
		WeightG: 16.4, Protocol: "Digital", WidthMm: 25,
	}); err != nil { panic(err) }
	if err := pb.CreateEsc(ctx, tx, &pb.Esc{
		Uuid: "043313ce-9ddb-47df-b915-56a28855f49e", Id: "happymodel-x12-5-in-1-aio", Manufacturer: "Happymodel", Model: "Happymodel X12 5-IN-1 AIO",
		WeightG: 5.1, ContinuousAmps: 12, BurstAmps: 15, MaxMotors: 4, Firmware: "BLHeli_32",
	}); err != nil { panic(err) }
	if err := pb.CreateFrame(ctx, tx, &pb.Frame{
		Uuid: "923b27f5-5b9f-47d4-9920-98c5477bc577", Id: "happymodel-mobula8-frame", Manufacturer: "Happymodel", Model: "Happymodel Mobula8 Frame",
		WeightG: 11.5, WheelbaseMm: 85.0, MaxPropSizeInches: 2.01,
		Geometry: "True-X",
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "21f70134-462c-4933-b508-a14b9f84ac51", Id: "happymodel-ex1103-8000kv", Manufacturer: "Happymodel", Model: "Happymodel EX1103 8000KV",
		WeightG: 3.8, StatorDiameterMm: 11.0, StatorHeightMm: 3.0, Kv: 8000,
	}); err != nil { panic(err) }
	if err := pb.CreateFrame(ctx, tx, &pb.Frame{
		Uuid: "0e06afa6-20af-4e2e-952c-d60df3d21846", Id: "flywoo-explorer-lr-4-frame", Manufacturer: "Flywoo", Model: "Flywoo Explorer LR 4 Frame",
		WeightG: 41.0, WheelbaseMm: 168.0, MaxPropSizeInches: 4.0,
		Geometry: "True-X",
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "8918b789-fa95-4e40-8fb4-4d71fbd9c0ef", Id: "flywoo-robo-1002-15500kv", Manufacturer: "Flywoo", Model: "Flywoo ROBO 1002 15500KV",
		WeightG: 2.5, StatorDiameterMm: 10.0, StatorHeightMm: 2.0, Kv: 15500,
	}); err != nil { panic(err) }
	if err := pb.CreateFrame(ctx, tx, &pb.Frame{
		Uuid: "322f0ecd-c122-4eb8-b709-5325247be75c", Id: "betafpv-meteor75-pro-frame", Manufacturer: "BetaFPV", Model: "BetaFPV Meteor75 Pro Frame",
		WeightG: 5.8, WheelbaseMm: 80.8, MaxPropSizeInches: 1.77,
		Geometry: "True-X",
	}); err != nil { panic(err) }
	if err := pb.CreateEsc(ctx, tx, &pb.Esc{
		Uuid: "41135f85-6b65-4ccd-a019-9d9bd1973294", Id: "betafpv-f4-1s-5a-aio", Manufacturer: "BetaFPV", Model: "BetaFPV F4 1S 5A AIO",
		WeightG: 2.96, ContinuousAmps: 5, BurstAmps: 6, MaxMotors: 4, Firmware: "BLHeli_32",
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "6f8cd304-aefd-4836-b82f-6a0c7b544662", Id: "betafpv-0802se-19500kv", Manufacturer: "BetaFPV", Model: "BetaFPV 0802SE 19500KV",
		WeightG: 1.83, StatorDiameterMm: 8.0, StatorHeightMm: 2.0, Kv: 19500,
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "b3149f97-87e5-434b-9bd8-82fcd7b86585", Id: "betafpv-lava-1s-300mah", Manufacturer: "BetaFPV", Model: "BetaFPV Lava 1S 300mAh",
		WeightG: 8.3, CapacityMah: 300, CellCountS: 1,
		Chemistry: "LiPo", Connector: "BT2.0",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "43541b0f-2247-40ae-ac02-63cd7354973a", Id: "betafpv-lava-2s-450mah", Manufacturer: "BetaFPV", Model: "BetaFPV Lava 2S 450mAh",
		WeightG: 26.1, CapacityMah: 450, CellCountS: 2,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "d91e3ca4-051c-40a0-b0e1-1a3a9c4b9dfd", Id: "gnb-3s-650mah-80c", Manufacturer: "GNB", Model: "GNB 3S 650mAh 80C",
		WeightG: 45.9, CapacityMah: 650, CellCountS: 3,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "1fa0c6b7-496b-4cbe-817e-a5052ffc4e66", Id: "ovonic-4s-1300mah-100c", Manufacturer: "Ovonic", Model: "Ovonic 4S 1300mAh 100C",
		WeightG: 156.0, CapacityMah: 1300, CellCountS: 4,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "d427489a-683b-461c-82b7-081cbb4d8ba4", Id: "tattu-r-line-version-5.0-1400mah-6s", Manufacturer: "Tattu", Model: "Tattu R-Line Version 5.0 1400mAh 6S",
		WeightG: 222.0, CapacityMah: 1400, CellCountS: 6,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "03d2ced4-8fb7-48f3-ac22-28a8b2af3245", Id: "gnb-8s-1100mah-130c", Manufacturer: "GNB", Model: "GNB 8S 1100mAh 130C",
		WeightG: 256.7, CapacityMah: 1100, CellCountS: 8,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "bac19aec-99fc-43d4-b96d-d1feb30e6b6e", Id: "gemfan-hurricane-51466-v2", Manufacturer: "Gemfan", Model: "Gemfan Hurricane 51466 V2",
		WeightG: 4.2, DiameterInches: 5.19, PitchInches: 3.6, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "0801832b-b458-4ad5-a311-0e4f1efcc8da", Id: "gemfan-floppy-proppy-f5135", Manufacturer: "Gemfan", Model: "Gemfan Floppy Proppy F5135",
		WeightG: 4.7, DiameterInches: 5.1, PitchInches: 3.5, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "85fd29c1-857d-48e0-a5f7-f454ad4c52b6", Id: "hqprop-ethix-s5-light-grey", Manufacturer: "HQProp", Model: "HQProp Ethix S5 Light Grey",
		WeightG: 3.7, DiameterInches: 5.0, PitchInches: 4.0, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "cd0d879e-7b30-4d22-b4cd-475f103b8b70", Id: "hqprop-7x4x3", Manufacturer: "HQProp", Model: "HQProp 7X4X3",
		WeightG: 9.1, DiameterInches: 7.0, PitchInches: 4.0, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "426b4655-788f-4642-a1b0-33adbed68dcc", Id: "betafpv-lava-1s-450mah", Manufacturer: "BetaFPV", Model: "BetaFPV LAVA 1S 450mAh",
		WeightG: 12.2, CapacityMah: 450, CellCountS: 1,
		Chemistry: "LiPo", Connector: "BT2.0",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "221677db-5f86-42ce-9c48-02525e7c5e0b", Id: "betafpv-lava-1s-550mah", Manufacturer: "BetaFPV", Model: "BetaFPV LAVA 1S 550mAh",
		WeightG: 14.0, CapacityMah: 550, CellCountS: 1,
		Chemistry: "LiPo", Connector: "BT2.0",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "b4095c95-d15e-49c9-9ed0-bbcc518a0787", Id: "betafpv-lava-3s-450mah", Manufacturer: "BetaFPV", Model: "BetaFPV LAVA 3S 450mAh",
		WeightG: 37.8, CapacityMah: 450, CellCountS: 3,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "b64e9d40-519c-489c-9e48-60e57dfd70e8", Id: "betafpv-lava-4s-450mah", Manufacturer: "BetaFPV", Model: "BetaFPV LAVA 4S 450mAh",
		WeightG: 49.4, CapacityMah: 450, CellCountS: 4,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "759b6c4a-b1c2-4684-b65b-8659b8fde102", Id: "betafpv-lava-4s-850mah", Manufacturer: "BetaFPV", Model: "BetaFPV LAVA 4S 850mAh",
		WeightG: 98.0, CapacityMah: 850, CellCountS: 4,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "cb573ed3-63e0-45f5-8602-b7ceea8237ea", Id: "tattu-r-line-4s-650mah", Manufacturer: "Tattu", Model: "Tattu R-Line 4S 650mAh",
		WeightG: 82.0, CapacityMah: 650, CellCountS: 4,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "d33e58f9-7089-45c0-8141-d9ecd23ad039", Id: "tattu-r-line-4s-1300mah", Manufacturer: "Tattu", Model: "Tattu R-Line 4S 1300mAh",
		WeightG: 150.0, CapacityMah: 1300, CellCountS: 4,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "5bc7132a-e7b9-4814-8f62-4d935619c624", Id: "tattu-r-line-4s-1550mah", Manufacturer: "Tattu", Model: "Tattu R-Line 4S 1550mAh",
		WeightG: 177.0, CapacityMah: 1550, CellCountS: 4,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "2afad43f-68b3-4452-9078-26ace845eacb", Id: "tattu-r-line-6s-1050mah", Manufacturer: "Tattu", Model: "Tattu R-Line 6S 1050mAh",
		WeightG: 180.0, CapacityMah: 1050, CellCountS: 6,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "4b26b55e-2df5-43f9-93b1-637f5e40726c", Id: "tattu-r-line-6s-1300mah", Manufacturer: "Tattu", Model: "Tattu R-Line 6S 1300mAh",
		WeightG: 208.0, CapacityMah: 1300, CellCountS: 6,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "d76f893c-7fc8-4882-aaec-b8e3a27734fa", Id: "ovonic-3s-450mah", Manufacturer: "Ovonic", Model: "Ovonic 3S 450mAh",
		WeightG: 33.6, CapacityMah: 450, CellCountS: 3,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "cf84db8e-1d21-47a8-8172-9dcf8c3ca161", Id: "ovonic-3s-850mah", Manufacturer: "Ovonic", Model: "Ovonic 3S 850mAh",
		WeightG: 76.0, CapacityMah: 850, CellCountS: 3,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "762c48ce-f399-4060-8966-6d61621d94f0", Id: "ovonic-4s-850mah", Manufacturer: "Ovonic", Model: "Ovonic 4S 850mAh",
		WeightG: 102.0, CapacityMah: 850, CellCountS: 4,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "cb0574cd-8198-4751-9f8a-fe8f5aef6953", Id: "ovonic-4s-1550mah", Manufacturer: "Ovonic", Model: "Ovonic 4S 1550mAh",
		WeightG: 184.0, CapacityMah: 1550, CellCountS: 4,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "073795ac-70cb-4239-ad27-5e192718ca1f", Id: "ovonic-6s-1000mah", Manufacturer: "Ovonic", Model: "Ovonic 6S 1000mAh",
		WeightG: 171.0, CapacityMah: 1000, CellCountS: 6,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "f5ecf3a8-852e-42f4-aedb-6823fdbe143d", Id: "ovonic-6s-1200mah", Manufacturer: "Ovonic", Model: "Ovonic 6S 1200mAh",
		WeightG: 195.0, CapacityMah: 1200, CellCountS: 6,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "662fcdf9-8f1e-44e3-b2df-63a33fc3dd38", Id: "gnb-1s-380mah", Manufacturer: "GNB", Model: "GNB 1S 380mAh",
		WeightG: 10.0, CapacityMah: 380, CellCountS: 1,
		Chemistry: "LiPo", Connector: "BT2.0",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "f4657c28-15c0-4ee2-b8d3-987f76bf14a3", Id: "gnb-1s-530mah", Manufacturer: "GNB", Model: "GNB 1S 530mAh",
		WeightG: 12.0, CapacityMah: 530, CellCountS: 1,
		Chemistry: "LiPo", Connector: "BT2.0",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "d8e63146-9249-4006-9fc1-a6a315a2bb19", Id: "gnb-2s-530mah", Manufacturer: "GNB", Model: "GNB 2S 530mAh",
		WeightG: 26.5, CapacityMah: 530, CellCountS: 2,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "72ea4e93-4512-4c27-a6d0-24d8831f1a3f", Id: "gnb-4s-650mah", Manufacturer: "GNB", Model: "GNB 4S 650mAh",
		WeightG: 70.0, CapacityMah: 650, CellCountS: 4,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "6e7121a7-9b08-4b5c-9a6f-30f1cadfdb39", Id: "gnb-4s-1100mah", Manufacturer: "GNB", Model: "GNB 4S 1100mAh",
		WeightG: 133.0, CapacityMah: 1100, CellCountS: 4,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "839eff53-628c-4f37-8393-d146c61f289f", Id: "gnb-6s-1100mah", Manufacturer: "GNB", Model: "GNB 6S 1100mAh",
		WeightG: 177.0, CapacityMah: 1100, CellCountS: 6,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "888d263a-b08c-4c9e-b1f4-d96552ffd682", Id: "gnb-6s-1300mah", Manufacturer: "GNB", Model: "GNB 6S 1300mAh",
		WeightG: 189.0, CapacityMah: 1300, CellCountS: 6,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "88f6cc91-3388-42e4-bce1-3be144082a57", Id: "betafpv-2s-300mah-(unverified)", Manufacturer: "BetaFPV", Model: "BetaFPV 2S 300mAh (Unverified)",
		WeightG: 15.0, CapacityMah: 300, CellCountS: 2,
		Chemistry: "LiPo", Connector: "BT2.0",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "8155a9c6-38a4-45bb-90bd-0991f74a99d0", Id: "betafpv-lava-2s-550mah", Manufacturer: "BetaFPV", Model: "BetaFPV LAVA 2S 550mAh",
		WeightG: 29.5, CapacityMah: 550, CellCountS: 2,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "b3bdf2b8-ea20-4073-8f98-63fe39fe5205", Id: "tattu-2s-450mah", Manufacturer: "Tattu", Model: "Tattu 2S 450mAh",
		WeightG: 29.0, CapacityMah: 450, CellCountS: 2,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "2c6dd4bc-9ab1-4062-b6d1-56adfab5a1b9", Id: "tattu-2s-850mah", Manufacturer: "Tattu", Model: "Tattu 2S 850mAh",
		WeightG: 40.0, CapacityMah: 850, CellCountS: 2,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "e6c4fde3-d201-4655-b901-6f7321de0518", Id: "ovonic-2s-450mah-(unverified)", Manufacturer: "Ovonic", Model: "Ovonic 2S 450mAh (Unverified)",
		WeightG: 24.0, CapacityMah: 450, CellCountS: 2,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "01724ebd-029f-447a-84dc-0e17bf421296", Id: "ovonic-2s-850mah-(unverified)", Manufacturer: "Ovonic", Model: "Ovonic 2S 850mAh (Unverified)",
		WeightG: 45.0, CapacityMah: 850, CellCountS: 2,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "1c8782e6-5a3c-45ee-b273-19c42bd94dfa", Id: "gnb-2s-380mah", Manufacturer: "GNB", Model: "GNB 2S 380mAh",
		WeightG: 22.0, CapacityMah: 380, CellCountS: 2,
		Chemistry: "LiPo", Connector: "BT2.0",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "e28ab1f7-4eb6-44af-9154-01fa57e58ff2", Id: "gnb-2s-450mah", Manufacturer: "GNB", Model: "GNB 2S 450mAh",
		WeightG: 27.0, CapacityMah: 450, CellCountS: 2,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "3dff8464-4987-4fea-aa2d-0dedd8399bc4", Id: "betafpv-2s-850mah-(unverified)", Manufacturer: "BetaFPV", Model: "BetaFPV 2S 850mAh (Unverified)",
		WeightG: 45.0, CapacityMah: 850, CellCountS: 2,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "43d224c1-c836-4080-8023-21bf320ecd9e", Id: "tattu-r-line-4s-1800mah", Manufacturer: "Tattu", Model: "Tattu R-Line 4S 1800mAh",
		WeightG: 197.0, CapacityMah: 1800, CellCountS: 4,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "70d12b53-9235-40f1-808f-1ae01315e6b3", Id: "tattu-r-line-5s-1300mah-(unverified)", Manufacturer: "Tattu", Model: "Tattu R-Line 5S 1300mAh (Unverified)",
		WeightG: 185.0, CapacityMah: 1300, CellCountS: 5,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "ce4f34ef-29fe-48ef-afba-22536c9c2e4b", Id: "tattu-r-line-5s-1550mah", Manufacturer: "Tattu", Model: "Tattu R-Line 5S 1550mAh",
		WeightG: 214.0, CapacityMah: 1550, CellCountS: 5,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "beb448f8-965d-4647-880c-a468f573be9c", Id: "tattu-r-line-6s-1250mah-(unverified)", Manufacturer: "Tattu", Model: "Tattu R-Line 6S 1250mAh (Unverified)",
		WeightG: 200.0, CapacityMah: 1250, CellCountS: 6,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "93e4fab4-ecde-4083-8d9f-85d4dddbf485", Id: "tattu-r-line-6s-1550mah", Manufacturer: "Tattu", Model: "Tattu R-Line 6S 1550mAh",
		WeightG: 254.0, CapacityMah: 1550, CellCountS: 6,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "eed490f2-be45-4aa8-a08a-34e0e0a809bb", Id: "ovonic-2s-650mah", Manufacturer: "Ovonic", Model: "Ovonic 2S 650mAh",
		WeightG: 45.0, CapacityMah: 650, CellCountS: 2,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "409f74fb-e67f-4826-b77a-2d37459715b2", Id: "ovonic-3s-1300mah", Manufacturer: "Ovonic", Model: "Ovonic 3S 1300mAh",
		WeightG: 115.0, CapacityMah: 1300, CellCountS: 3,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "7481d1ba-36ae-4400-9902-39dca13d5b54", Id: "ovonic-4s-1800mah", Manufacturer: "Ovonic", Model: "Ovonic 4S 1800mAh",
		WeightG: 205.0, CapacityMah: 1800, CellCountS: 4,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "4a032ce6-04fc-4637-b6d6-aec513a42f32", Id: "ovonic-6s-1550mah", Manufacturer: "Ovonic", Model: "Ovonic 6S 1550mAh",
		WeightG: 245.0, CapacityMah: 1550, CellCountS: 6,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "d954c8cd-9ff2-4367-8431-fbc12ead778e", Id: "gnb-1s-650mah", Manufacturer: "GNB", Model: "GNB 1S 650mAh",
		WeightG: 17.0, CapacityMah: 650, CellCountS: 1,
		Chemistry: "LiPo", Connector: "BT2.0",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "f6d83de8-c91f-466f-8c78-71d03adb1871", Id: "gnb-1s-720mah", Manufacturer: "GNB", Model: "GNB 1S 720mAh",
		WeightG: 16.5, CapacityMah: 720, CellCountS: 1,
		Chemistry: "LiPo", Connector: "BT2.0",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "b9de474f-0998-4a43-a6de-28fc775743b3", Id: "gnb-2s-650mah", Manufacturer: "GNB", Model: "GNB 2S 650mAh",
		WeightG: 31.4, CapacityMah: 650, CellCountS: 2,
		Chemistry: "LiPo", Connector: "XT30",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "b26d4204-233c-4ed2-98ad-36370742dbbc", Id: "gnb-3s-1100mah", Manufacturer: "GNB", Model: "GNB 3S 1100mAh",
		WeightG: 111.0, CapacityMah: 1100, CellCountS: 3,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "01f45a03-4009-496e-9c70-d0697fe302fa", Id: "gnb-4s-1550mah", Manufacturer: "GNB", Model: "GNB 4S 1550mAh",
		WeightG: 189.0, CapacityMah: 1550, CellCountS: 4,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "519db6a0-f594-4759-a5bd-285f8bfd078a", Id: "gnb-6s-1550mah", Manufacturer: "GNB", Model: "GNB 6S 1550mAh",
		WeightG: 273.0, CapacityMah: 1550, CellCountS: 6,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreateBattery(ctx, tx, &pb.Battery{
		Uuid: "c6d88b3a-67a9-461d-aa1d-e6047bce5ec6", Id: "gnb-8s-1300mah", Manufacturer: "GNB", Model: "GNB 8S 1300mAh",
		WeightG: 252.0, CapacityMah: 1300, CellCountS: 8,
		Chemistry: "LiPo", Connector: "XT60",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "a609a8fc-e509-4f5e-a3b5-2ee4847192cf", Id: "gemfan-1219-3-blade", Manufacturer: "Gemfan", Model: "Gemfan 1219 3-Blade",
		WeightG: 0.28, DiameterInches: 1.22, PitchInches: 1.89, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "1d421d54-2b93-4bfe-a626-b1f3918f3426", Id: "gemfan-1636-4-blade", Manufacturer: "Gemfan", Model: "Gemfan 1636 4-Blade",
		WeightG: 0.5, DiameterInches: 1.57, PitchInches: 3.6, Blades: 4, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "db4c27cd-dbf6-4fb8-83d1-5d00ee86562a", Id: "gemfan-2023-3-blade", Manufacturer: "Gemfan", Model: "Gemfan 2023 3-Blade",
		WeightG: 0.88, DiameterInches: 2.0, PitchInches: 2.3, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "9b6afc05-af45-45c1-9171-6c08de3d8c8d", Id: "gemfan-3016-3-blade", Manufacturer: "Gemfan", Model: "Gemfan 3016 3-Blade",
		WeightG: 1.18, DiameterInches: 3.0, PitchInches: 1.6, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "f438ea4d-29f6-4730-b5d6-105105d04294", Id: "gemfan-4023-3-blade", Manufacturer: "Gemfan", Model: "Gemfan 4023 3-Blade",
		WeightG: 1.9, DiameterInches: 4.0, PitchInches: 2.3, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "c6ccaf42-c038-463b-b311-1e53a75c2fa0", Id: "gemfan-hurricane-51433", Manufacturer: "Gemfan", Model: "Gemfan Hurricane 51433",
		WeightG: 3.8, DiameterInches: 5.15, PitchInches: 4.33, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "81ff33d3-02da-4d1f-a258-0974cd1aaf38", Id: "gemfan-hurricane-51499", Manufacturer: "Gemfan", Model: "Gemfan Hurricane 51499",
		WeightG: 5.1, DiameterInches: 5.15, PitchInches: 4.99, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "a84c2809-0a8d-4eff-b7f3-ac4e36586e3b", Id: "gemfan-7040-3-blade", Manufacturer: "Gemfan", Model: "Gemfan 7040 3-Blade",
		WeightG: 8.7, DiameterInches: 7.0, PitchInches: 4.0, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "8a9d1fa6-3a22-4bed-86e8-2352ec189f35", Id: "hqprop-micro-whoop-31x3", Manufacturer: "HQProp", Model: "HQProp Micro Whoop 31x3",
		WeightG: 0.27, DiameterInches: 1.22, PitchInches: 1.18, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "9c6cb944-f7cd-4459-a3c9-b288309e88fa", Id: "hqprop-micro-whoop-40x4", Manufacturer: "HQProp", Model: "HQProp Micro Whoop 40x4",
		WeightG: 0.5, DiameterInches: 1.57, PitchInches: 1.57, Blades: 4, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "d00fa7e1-7efc-4057-ba75-e0d6d9b84b3b", Id: "hqprop-t65mm-2.5-inch", Manufacturer: "HQProp", Model: "HQProp T65MM 2.5 Inch",
		WeightG: 1.2, DiameterInches: 2.56, PitchInches: 1.57, Blades: 2, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "50f7726c-df27-4037-8bcb-c1779dac76f9", Id: "hqprop-t3x3x3", Manufacturer: "HQProp", Model: "HQProp T3x3x3",
		WeightG: 1.5, DiameterInches: 3.0, PitchInches: 3.0, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "89594ace-b5b9-4494-91c3-de6d839e77b7", Id: "hqprop-t4x2.5x2", Manufacturer: "HQProp", Model: "HQProp T4x2.5x2",
		WeightG: 1.4, DiameterInches: 4.0, PitchInches: 2.5, Blades: 2, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "d1ce02e4-961c-499d-8424-3b84424edfde", Id: "hqprop-ethix-p3-peanut-butter", Manufacturer: "HQProp", Model: "HQProp Ethix P3 Peanut Butter",
		WeightG: 3.8, DiameterInches: 5.1, PitchInches: 3.0, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "6d5540a7-7f07-40a5-840d-185c39f77038", Id: "hqprop-ethix-s3-watermelon", Manufacturer: "HQProp", Model: "HQProp Ethix S3 Watermelon",
		WeightG: 3.6, DiameterInches: 5.0, PitchInches: 3.1, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "180c5d56-8970-405b-9308-254d1bde06de", Id: "hqprop-5.1x4.1x3", Manufacturer: "HQProp", Model: "HQProp 5.1x4.1x3",
		WeightG: 4.7, DiameterInches: 5.1, PitchInches: 4.1, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "194a58be-4a4c-48b3-bc42-85a97d25074a", Id: "hqprop-7x3.5x3", Manufacturer: "HQProp", Model: "HQProp 7x3.5x3",
		WeightG: 8.8, DiameterInches: 7.0, PitchInches: 3.5, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreateFlightController(ctx, tx, &pb.FlightController{
		Uuid: "b2337734-fe18-42a2-a1fa-c190bc9488c1", Id: "hglrc-zeus-f722", Manufacturer: "HGLRC", Model: "HGLRC Zeus F722",
		WeightG: 8.1, Processor: "STM32F722RET6", Gyro: "MPU6000",
	}); err != nil { panic(err) }
	if err := pb.CreateEsc(ctx, tx, &pb.Esc{
		Uuid: "19578f9c-b43e-4323-b6ee-8af447a274f4", Id: "hglrc-zeus-48a", Manufacturer: "HGLRC", Model: "HGLRC Zeus 48A",
		WeightG: 18.3, ContinuousAmps: 48, BurstAmps: 55, MaxMotors: 4, Firmware: "BLHeli_32",
	}); err != nil { panic(err) }
	if err := pb.CreateFrame(ctx, tx, &pb.Frame{
		Uuid: "1050cda5-594c-4737-bafd-965b75e2f3e3", Id: "hglrc-sector-d5-frame", Manufacturer: "HGLRC", Model: "HGLRC Sector D5 Frame",
		WeightG: 121.7, WheelbaseMm: 225.0, MaxPropSizeInches: 5.0,
		Geometry: "True-X",
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "0f7ca081-eec2-430b-9986-7295222c055c", Id: "hglrc-aeolus-2105.5-2800kv", Manufacturer: "HGLRC", Model: "HGLRC Aeolus 2105.5 2800KV",
		WeightG: 22.3, StatorDiameterMm: 0.0, StatorHeightMm: 0.0, Kv: 2800,
	}); err != nil { panic(err) }
	if err := pb.CreateVideoTransmitter(ctx, tx, &pb.VideoTransmitter{
		Uuid: "73be0eb6-e9cf-4209-b809-7dad5c0e5ae9", Id: "hglrc-zeus-800mw-vtx", Manufacturer: "HGLRC", Model: "HGLRC Zeus 800mW VTX",
		WeightG: 4.8, Protocol: "", MaxPowerMw: 800,
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "c049c56b-c320-4e33-98aa-bac2f85822e1", Id: "hglrc-specter-2306.5-1900kv", Manufacturer: "HGLRC", Model: "HGLRC Specter 2306.5 1900KV",
		WeightG: 34.0, StatorDiameterMm: 0.0, StatorHeightMm: 0.0, Kv: 1900,
	}); err != nil { panic(err) }
	if err := pb.CreateFlightController(ctx, tx, &pb.FlightController{
		Uuid: "971706b1-9710-4c6a-86d3-26db8b70d0ba", Id: "hglrc-zeus-f722-mini", Manufacturer: "HGLRC", Model: "HGLRC Zeus F722 Mini",
		WeightG: 4.6, Processor: "STM32F722", Gyro: "MPU6000",
	}); err != nil { panic(err) }
	if err := pb.CreateFlightController(ctx, tx, &pb.FlightController{
		Uuid: "15a59535-5041-4378-a0a6-716dff27bf88", Id: "hglrc-specter-f405-v2", Manufacturer: "HGLRC", Model: "HGLRC Specter F405 V2",
		WeightG: 7.2, Processor: "STM32F405", Gyro: "MPU6000",
	}); err != nil { panic(err) }
	if err := pb.CreateEsc(ctx, tx, &pb.Esc{
		Uuid: "0ceda915-f01c-4b27-bc8c-e01c22e58d2a", Id: "hglrc-specter-60a-4-in-1-esc", Manufacturer: "HGLRC", Model: "HGLRC Specter 60A 4-in-1 ESC",
		WeightG: 19.5, ContinuousAmps: 60, BurstAmps: 65, MaxMotors: 4, Firmware: "BLHeli_32",
	}); err != nil { panic(err) }
	if err := pb.CreateFrame(ctx, tx, &pb.Frame{
		Uuid: "ce4e0a71-8430-4fa4-871e-18447e2f882a", Id: "hglrc-rekon3-frame", Manufacturer: "HGLRC", Model: "HGLRC Rekon3 Frame",
		WeightG: 23.0, WheelbaseMm: 140.0, MaxPropSizeInches: 3.0,
		Geometry: "True-X",
	}); err != nil { panic(err) }
	if err := pb.CreateFrame(ctx, tx, &pb.Frame{
		Uuid: "f19553c3-6e78-4752-a957-9abe978c09f6", Id: "hglrc-rekon5-frame", Manufacturer: "HGLRC", Model: "HGLRC Rekon5 Frame",
		WeightG: 72.0, WheelbaseMm: 210.0, MaxPropSizeInches: 5.0,
		Geometry: "True-X",
	}); err != nil { panic(err) }
	if err := pb.CreateFrame(ctx, tx, &pb.Frame{
		Uuid: "0c25c3b7-ab22-4b13-b8fe-708a83898c3b", Id: "hglrc-sector-x5-frame", Manufacturer: "HGLRC", Model: "HGLRC Sector X5 Frame",
		WeightG: 120.0, WheelbaseMm: 210.0, MaxPropSizeInches: 5.0,
		Geometry: "True-X",
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "c1b1f0f5-881f-4370-b27f-c5f1e4825439", Id: "hglrc-aeolus-1102-18000kv", Manufacturer: "HGLRC", Model: "HGLRC Aeolus 1102 18000KV",
		WeightG: 3.2, StatorDiameterMm: 11.0, StatorHeightMm: 2.0, Kv: 18000,
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "59da674f-dbc5-4787-a7ee-6eb365b214f1", Id: "hglrc-aeolus-1404-2800kv", Manufacturer: "HGLRC", Model: "HGLRC Aeolus 1404 2800KV",
		WeightG: 11.0, StatorDiameterMm: 14.0, StatorHeightMm: 4.0, Kv: 2800,
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "a97c55b6-0ac7-4762-aa95-15b25c09a6a4", Id: "hglrc-aeolus-2004-3000kv", Manufacturer: "HGLRC", Model: "HGLRC Aeolus 2004 3000KV",
		WeightG: 16.6, StatorDiameterMm: 20.0, StatorHeightMm: 4.0, Kv: 3000,
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "149b635a-b27c-4f7b-a382-47df3b4dd9c0", Id: "hglrc-specter-1804-3450kv", Manufacturer: "HGLRC", Model: "HGLRC Specter 1804 3450KV",
		WeightG: 13.0, StatorDiameterMm: 18.0, StatorHeightMm: 4.0, Kv: 3500,
	}); err != nil { panic(err) }
	if err := pb.CreateVideoTransmitter(ctx, tx, &pb.VideoTransmitter{
		Uuid: "74eb2fce-87ca-4f64-bd62-dfa0fa02b96c", Id: "hglrc-zeus-nano-vtx-350mw", Manufacturer: "HGLRC", Model: "HGLRC Zeus Nano VTX 350mW",
		WeightG: 2.4, Protocol: "", MaxPowerMw: 350,
	}); err != nil { panic(err) }
	if err := pb.CreateReceiver(ctx, tx, &pb.Receiver{
		Uuid: "42e4ac1f-d5e8-4863-9eeb-f0d8ad8eeffd", Id: "hglrc-hermes-expresslrs-2.4ghz", Manufacturer: "HGLRC", Model: "HGLRC Hermes ExpressLRS 2.4GHz",
		WeightG: 0.8, Protocol: "ExpressLRS (ELRS)", FrequencyBandGhz: 2.4,
	}); err != nil { panic(err) }
	if err := pb.CreateReceiver(ctx, tx, &pb.Receiver{
		Uuid: "d1a50985-ef0b-4357-b944-7367b21e24f5", Id: "hglrc-hermes-expresslrs-900mhz", Manufacturer: "HGLRC", Model: "HGLRC Hermes ExpressLRS 900MHz",
		WeightG: 0.8, Protocol: "ExpressLRS (ELRS)", FrequencyBandGhz: 2.4,
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "a4ae9a61-9674-4622-8e3f-06f2f00c5976", Id: "hglrc-specter-1202.5-11000kv", Manufacturer: "HGLRC", Model: "HGLRC Specter 1202.5 11000KV",
		WeightG: 4.6, StatorDiameterMm: 0.0, StatorHeightMm: 0.0, Kv: 11000,
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "92685b6e-38d0-4a20-96b5-3f8359f13016", Id: "hglrc-aeolus-1303.5-2500kv", Manufacturer: "HGLRC", Model: "HGLRC Aeolus 1303.5 2500KV",
		WeightG: 6.5, StatorDiameterMm: 0.0, StatorHeightMm: 0.0, Kv: 2500,
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "701c1649-c31e-48c6-aef4-4daeb76abd29", Id: "hglrc-aeolus-2207.5-2550kv", Manufacturer: "HGLRC", Model: "HGLRC Aeolus 2207.5 2550KV",
		WeightG: 33.0, StatorDiameterMm: 0.0, StatorHeightMm: 0.0, Kv: 2550,
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "dd3137e4-e8f9-40d3-92db-af552417897e", Id: "hglrc-aeolus-2306.5-2550kv", Manufacturer: "HGLRC", Model: "HGLRC Aeolus 2306.5 2550KV",
		WeightG: 37.0, StatorDiameterMm: 0.0, StatorHeightMm: 0.0, Kv: 2550,
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "6275a11b-96dd-4c39-a464-9efa476dcc75", Id: "hglrc-aeolus-2806.5-1250kv", Manufacturer: "HGLRC", Model: "HGLRC Aeolus 2806.5 1250KV",
		WeightG: 47.0, StatorDiameterMm: 0.0, StatorHeightMm: 0.0, Kv: 1250,
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "0f76a74f-ab6f-4f19-a96b-6e5cb0d192de", Id: "hglrc-specter-1003-10000kv", Manufacturer: "HGLRC", Model: "HGLRC Specter 1003 10000KV",
		WeightG: 3.35, StatorDiameterMm: 10.0, StatorHeightMm: 3.0, Kv: 10000,
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "3f6ac5f9-924a-41b0-a205-bbd5dd9b85b0", Id: "t-motor-m1103-8000kv", Manufacturer: "T-Motor", Model: "T-Motor M1103 8000KV",
		WeightG: 3.4, StatorDiameterMm: 11.0, StatorHeightMm: 3.0, Kv: 8000,
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "3c7d91b3-b937-4111-af63-92001e6fc7dd", Id: "hglrc-specter-1404-2750kv", Manufacturer: "HGLRC", Model: "HGLRC Specter 1404 2750KV",
		WeightG: 10.0, StatorDiameterMm: 14.0, StatorHeightMm: 4.0, Kv: 2750,
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "22eb1c59-a075-4b0e-9960-4bafc7871c01", Id: "hglrc-specter-2004-3000kv", Manufacturer: "HGLRC", Model: "HGLRC Specter 2004 3000KV",
		WeightG: 16.5, StatorDiameterMm: 20.0, StatorHeightMm: 4.0, Kv: 1800,
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "4464d837-4c9b-4be7-801c-645256ce8469", Id: "hglrc-specter-2207.5-1900kv", Manufacturer: "HGLRC", Model: "HGLRC Specter 2207.5 1900KV",
		WeightG: 34.0, StatorDiameterMm: 0.0, StatorHeightMm: 0.0, Kv: 1900,
	}); err != nil { panic(err) }
	if err := pb.CreateMotor(ctx, tx, &pb.Motor{
		Uuid: "6d1176e5-d9c4-46fc-a809-bd4e1e11c246", Id: "hglrc-specter-2806.5-1350kv", Manufacturer: "HGLRC", Model: "HGLRC Specter 2806.5 1350KV",
		WeightG: 51.9, StatorDiameterMm: 0.0, StatorHeightMm: 0.0, Kv: 1350,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "770f0c7f-51ef-475a-9835-92ac9361d5d4", Id: "lumenier-axii-2-5.8ghz-(rhcp)", Manufacturer: "Lumenier", Model: "Lumenier AXII 2 5.8GHz (RHCP)",
		WeightG: 7.8, Connector: "SMA", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "50f633cc-971a-4271-a515-71ed79a30c3e", Id: "truerc-singularity-5.8ghz-(rhcp)", Manufacturer: "TrueRC", Model: "TrueRC Singularity 5.8GHz (RHCP)",
		WeightG: 1.5, Connector: "U.FL", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "0020d8de-1de7-4fec-ae85-e0d84e9d14d4", Id: "foxeer-lollipop-4-(rhcp)", Manufacturer: "Foxeer", Model: "Foxeer Lollipop 4 (RHCP)",
		WeightG: 7.3, Connector: "SMA", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "2b441604-7264-4c23-aa43-fa12915cf226", Id: "foxeer-lollipop-4-micro-(rhcp)", Manufacturer: "Foxeer", Model: "Foxeer Lollipop 4 Micro (RHCP)",
		WeightG: 1.6, Connector: "U.FL", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "a27e48be-0c3c-4884-86a5-88ca80f447bc", Id: "tbs-triumph-pro-(rhcp)", Manufacturer: "TBS", Model: "TBS Triumph Pro (RHCP)",
		WeightG: 11.5, Connector: "SMA", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "772d1284-43de-4a48-9607-93bd735368cc", Id: "rush-cherry-5.8ghz-(rhcp)", Manufacturer: "Rush", Model: "Rush Cherry 5.8GHz (RHCP)",
		WeightG: 7.5, Connector: "SMA", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "75f229f4-b202-48ea-82d0-d17cb0e16df3", Id: "tbs-immortal-t-v2", Manufacturer: "TBS", Model: "TBS Immortal T V2",
		WeightG: 3.4, Connector: "U.FL", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "c74eb058-4a03-4c01-9b94-0cf434369dc2", Id: "tbs-diamond-antenna", Manufacturer: "TBS", Model: "TBS Diamond Antenna",
		WeightG: 20.6, Connector: "SMA", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "ab2ee340-028d-4bcc-9bdf-0663d625a3e7", Id: "radiomaster-2.4ghz-moxon", Manufacturer: "Radiomaster", Model: "Radiomaster 2.4GHz Moxon",
		WeightG: 12.0, Connector: "RP-SMA", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "46712ce9-e929-4dcc-a7ea-2827e5d61888", Id: "betafpv-moxon-antenna", Manufacturer: "BetaFPV", Model: "BetaFPV Moxon Antenna",
		WeightG: 12.0, Connector: "RP-SMA", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "5f13d7e0-3eec-419d-8d68-76d798604338", Id: "truerc-x-air-5.8-mk-ii-(rhcp)", Manufacturer: "TrueRC", Model: "TrueRC X-AIR 5.8 MK II (RHCP)",
		WeightG: 12.0, Connector: "SMA", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "bc138765-61b1-4e21-b306-ef6133e2f893", Id: "lumenier-axii-patch-antenna-(rhcp)", Manufacturer: "Lumenier", Model: "Lumenier AXII Patch Antenna (RHCP)",
		WeightG: 10.5, Connector: "SMA", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "8b2b12fc-b49d-46ae-88c0-31ee130a08e2", Id: "truerc-matchstick-5.8ghz-(rhcp)", Manufacturer: "TrueRC", Model: "TrueRC Matchstick 5.8GHz (RHCP)",
		WeightG: 7.8, Connector: "SMA", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "e8b0b1f9-0963-46b5-bd0d-683a8a75af5f", Id: "truerc-ocp-5.8ghz-(rhcp)", Manufacturer: "TrueRC", Model: "TrueRC OCP 5.8GHz (RHCP)",
		WeightG: 7.5, Connector: "SMA", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "8f707fa6-d8ab-40f5-8824-ea6971e62914", Id: "truerc-sniper-5.8ghz-(rhcp)", Manufacturer: "TrueRC", Model: "TrueRC Sniper 5.8GHz (RHCP)",
		WeightG: 15.0, Connector: "SMA", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "767b0b8b-18c3-4480-843b-653b2c87a569", Id: "lumenier-micro-axii-2-5.8ghz-(rhcp)", Manufacturer: "Lumenier", Model: "Lumenier Micro AXII 2 5.8GHz (RHCP)",
		WeightG: 1.4, Connector: "U.FL", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "5dc67fa6-82e8-43dd-8e00-38a096971d32", Id: "lumenier-double-axii-2-5.8ghz-(rhcp)", Manufacturer: "Lumenier", Model: "Lumenier Double AXII 2 5.8GHz (RHCP)",
		WeightG: 12.0, Connector: "SMA", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "c5135ef0-244a-4984-b471-82d20f5b3439", Id: "lumenier-axii-duo-5.8ghz-patch-(rhcp)", Manufacturer: "Lumenier", Model: "Lumenier AXII Duo 5.8GHz Patch (RHCP)",
		WeightG: 18.0, Connector: "SMA", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "b9747852-5b42-4351-9a9d-dfb2cc559afb", Id: "happymodel-2.4ghz-elrs-t-antenna", Manufacturer: "Happymodel", Model: "Happymodel 2.4GHz ELRS T-Antenna",
		WeightG: 1.2, Connector: "U.FL", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "3ad9ee47-38c0-4b71-8ca4-e690ea672870", Id: "happymodel-900mhz-elrs-t-antenna", Manufacturer: "Happymodel", Model: "Happymodel 900MHz ELRS T-Antenna",
		WeightG: 3.1, Connector: "U.FL", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "bdde4eed-7035-4a41-aa83-c0913393fe88", Id: "happymodel-5.8ghz-dipole-antenna", Manufacturer: "Happymodel", Model: "Happymodel 5.8GHz Dipole Antenna",
		WeightG: 0.8, Connector: "U.FL", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "5a823788-05aa-4471-887b-2068abe8e0ce", Id: "foxeer-lollipop-3-stubby-(rhcp)", Manufacturer: "Foxeer", Model: "Foxeer Lollipop 3 Stubby (RHCP)",
		WeightG: 4.8, Connector: "SMA", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "93dd1e46-71a6-48b9-ae51-677a377743f5", Id: "foxeer-echo-2-patch-antenna-(rhcp)", Manufacturer: "Foxeer", Model: "Foxeer Echo 2 Patch Antenna (RHCP)",
		WeightG: 11.5, Connector: "SMA", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "ed557d7e-a614-4aca-ae6b-72c9b2d08d19", Id: "foxeer-pagoda-pro-(rhcp)", Manufacturer: "Foxeer", Model: "Foxeer Pagoda Pro (RHCP)",
		WeightG: 11.0, Connector: "SMA", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "d4106767-bca1-4f2c-b83a-30149eadae34", Id: "lumenier-axii-2-(lhcp)", Manufacturer: "Lumenier", Model: "Lumenier AXII 2 (LHCP)",
		WeightG: 4.8, Connector: "SMA", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "b5404551-1455-4f7c-808a-fe99224c3883", Id: "lumenier-axii-2-5.8ghz-(lhcp)", Manufacturer: "Lumenier", Model: "Lumenier AXII 2 5.8GHz (LHCP)",
		WeightG: 7.8, Connector: "SMA", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "7d43a899-9302-4c06-8cdf-10f1e4cdec37", Id: "truerc-singularity-5.8ghz-(lhcp)", Manufacturer: "TrueRC", Model: "TrueRC Singularity 5.8GHz (LHCP)",
		WeightG: 1.5, Connector: "U.FL", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "2b56b985-df7c-4fb2-b857-c2bf9d20b42f", Id: "foxeer-lollipop-4-(lhcp)", Manufacturer: "Foxeer", Model: "Foxeer Lollipop 4 (LHCP)",
		WeightG: 7.3, Connector: "SMA", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "4bdf8de0-368e-4dcf-ac1b-a87cc263ba85", Id: "foxeer-lollipop-4-micro-(lhcp)", Manufacturer: "Foxeer", Model: "Foxeer Lollipop 4 Micro (LHCP)",
		WeightG: 1.6, Connector: "U.FL", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "de2555ad-3aca-474f-9304-1e73a3d0a100", Id: "tbs-triumph-pro-(lhcp)", Manufacturer: "TBS", Model: "TBS Triumph Pro (LHCP)",
		WeightG: 11.5, Connector: "SMA", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "2e2552ce-7f9d-4b4e-a29d-f8deebbc5244", Id: "rush-cherry-5.8ghz-(lhcp)", Manufacturer: "Rush", Model: "Rush Cherry 5.8GHz (LHCP)",
		WeightG: 7.5, Connector: "SMA", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "3ac9b7de-e8a1-4f95-9035-251ef3eafbf9", Id: "truerc-x-air-5.8-mk-ii-(lhcp)", Manufacturer: "TrueRC", Model: "TrueRC X-AIR 5.8 MK II (LHCP)",
		WeightG: 12.0, Connector: "SMA", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "62dbbe5f-ff0a-414c-b88d-915a38d14a05", Id: "lumenier-axii-patch-antenna-(lhcp)", Manufacturer: "Lumenier", Model: "Lumenier AXII Patch Antenna (LHCP)",
		WeightG: 10.5, Connector: "SMA", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "fbfde865-961b-4057-90bd-a9a9c721e128", Id: "truerc-matchstick-5.8ghz-(lhcp)", Manufacturer: "TrueRC", Model: "TrueRC Matchstick 5.8GHz (LHCP)",
		WeightG: 7.8, Connector: "SMA", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "387a8b39-8bc4-44a3-980b-535af4309aeb", Id: "truerc-ocp-5.8ghz-(lhcp)", Manufacturer: "TrueRC", Model: "TrueRC OCP 5.8GHz (LHCP)",
		WeightG: 7.5, Connector: "SMA", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "ca3080c1-64f2-487b-8fe0-29c7e6367660", Id: "truerc-sniper-5.8ghz-(lhcp)", Manufacturer: "TrueRC", Model: "TrueRC Sniper 5.8GHz (LHCP)",
		WeightG: 15.0, Connector: "SMA", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "c445e6bc-9018-4c18-a81e-1f3dbaca2630", Id: "lumenier-micro-axii-2-5.8ghz-(lhcp)", Manufacturer: "Lumenier", Model: "Lumenier Micro AXII 2 5.8GHz (LHCP)",
		WeightG: 1.4, Connector: "U.FL", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "6867dd7c-2e2c-4357-a020-e3b9cb4fcb58", Id: "lumenier-double-axii-2-5.8ghz-(lhcp)", Manufacturer: "Lumenier", Model: "Lumenier Double AXII 2 5.8GHz (LHCP)",
		WeightG: 12.0, Connector: "SMA", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "1e5e35d0-8097-476c-acd0-2a05ef88d9b3", Id: "lumenier-axii-duo-5.8ghz-patch-(lhcp)", Manufacturer: "Lumenier", Model: "Lumenier AXII Duo 5.8GHz Patch (LHCP)",
		WeightG: 18.0, Connector: "SMA", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "e3e23aea-e5a9-40bb-818e-3abe0ba3ceb1", Id: "foxeer-lollipop-3-stubby-(lhcp)", Manufacturer: "Foxeer", Model: "Foxeer Lollipop 3 Stubby (LHCP)",
		WeightG: 4.8, Connector: "SMA", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "af64bd15-c82d-4174-99c8-341bce6cfcf8", Id: "foxeer-echo-2-patch-antenna-(lhcp)", Manufacturer: "Foxeer", Model: "Foxeer Echo 2 Patch Antenna (LHCP)",
		WeightG: 11.5, Connector: "SMA", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "c0ce31d0-b000-4b36-bc6d-cca8daf29f0e", Id: "foxeer-pagoda-pro-(lhcp)", Manufacturer: "Foxeer", Model: "Foxeer Pagoda Pro (LHCP)",
		WeightG: 11.0, Connector: "SMA", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateCamera(ctx, tx, &pb.Camera{
		Uuid: "c966b786-ef4b-4c7b-81c2-c911a8790df8", Id: "caddx-ratel-2", Manufacturer: "Caddx", Model: "Caddx Ratel 2",
		WeightG: 5.9, Protocol: "CVBS", WidthMm: 19,
	}); err != nil { panic(err) }
	if err := pb.CreateFrame(ctx, tx, &pb.Frame{
		Uuid: "48abe2b3-1bf4-4380-8f26-75afc7f53089", Id: "sub250-oasis-frame", Manufacturer: "SUB250-Oasis", Model: "SUB250-Oasis Frame",
		WeightG: 53.0, WheelbaseMm: 135.0, MaxPropSizeInches: 2.5,
		Geometry: "True-X",
	}); err != nil { panic(err) }
	if err := pb.CreatePropeller(ctx, tx, &pb.Propeller{
		Uuid: "84d6959a-80c2-4e81-828e-71fee9065947", Id: "hqprop-2.5x2.5x3", Manufacturer: "HQProp", Model: "HQProp 2.5x2.5x3",
		WeightG: 1.5, DiameterInches: 2.5, PitchInches: 2.5, Blades: 3, Material: "Polycarbonate",
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "61f73529-a335-42e4-a7ee-4cf6f0306f48", Id: "tbs-tracer-immortal-t-antenna", Manufacturer: "TBS", Model: "TBS Tracer Immortal T Antenna",
		WeightG: 0.84, Connector: "U.FL", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "65c6a869-60f1-48cf-a75f-2cccbd971a0a", Id: "tbs-tracer-monopole-rx-antenna", Manufacturer: "TBS", Model: "TBS Tracer Monopole RX Antenna",
		WeightG: 0.33, Connector: "U.FL", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "d584739c-dc84-4769-9d23-76666fb6bc41", Id: "truerc-singularity-2.4-sma-(rhcp)", Manufacturer: "TrueRC", Model: "TrueRC Singularity 2.4 SMA (RHCP)",
		WeightG: 9.45, Connector: "SMA", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "4f52365d-f235-4198-8401-e13ef9aa2774", Id: "truerc-singularity-2.4-sma-(lhcp)", Manufacturer: "TrueRC", Model: "TrueRC Singularity 2.4 SMA (LHCP)",
		WeightG: 9.45, Connector: "SMA", Polarization: "LHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "9aee436e-77fc-4376-9fcc-ba69f8478330", Id: "truerc-d-pole-2.4-mk-ii", Manufacturer: "TrueRC", Model: "TrueRC D-POLE 2.4 MK II",
		WeightG: 2.0, Connector: "U.FL", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "4ee842aa-e740-4288-846c-a8d4242ae326", Id: "truerc-bardpole-2.4", Manufacturer: "TrueRC", Model: "TrueRC BARDpole 2.4",
		WeightG: 1.3, Connector: "U.FL", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "c4055518-1a21-4445-a760-becb19088b53", Id: "betafpv-2.4ghz-dipole-t-antenna", Manufacturer: "BETAFPV", Model: "BETAFPV 2.4GHz Dipole T-Antenna",
		WeightG: 1.2, Connector: "U.FL", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "e2d8da94-616c-499a-bbe5-f152f1a54b9d", Id: "tbs-crossfire-immortal-t-v2-(extra-extended)", Manufacturer: "TBS", Model: "TBS Crossfire Immortal T V2 (Extra Extended)",
		WeightG: 4.0, Connector: "U.FL", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "ee1b24de-21b5-4e69-b033-ac24e9b787e0", Id: "tbs-crossfire-stock-tx-v2-antenna", Manufacturer: "TBS", Model: "TBS Crossfire Stock Tx V2 Antenna",
		WeightG: 9.0, Connector: "SMA", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "6597d1ab-b031-4e3e-a764-8726543a6699", Id: "radiomaster-bandit-moxon-antenna", Manufacturer: "RadioMaster", Model: "RadioMaster Bandit Moxon Antenna",
		WeightG: 47.44, Connector: "RP-SMA", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "9eb07880-8f1b-4b7a-ae37-8456acf2dbfa", Id: "radiomaster-bandit-t-antenna", Manufacturer: "RadioMaster", Model: "RadioMaster Bandit T Antenna",
		WeightG: 14.0, Connector: "RP-SMA", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "9a17cb17-0275-4202-8811-74ee48aba608", Id: "radiomaster-ufl-915-868mhz-t-antenna", Manufacturer: "RadioMaster", Model: "RadioMaster UFL 915/868MHz T Antenna",
		WeightG: 2.02, Connector: "U.FL", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "2d2fac07-7a87-498c-a12b-e61da5332631", Id: "betafpv-moxon-antenna-(915mhz)", Manufacturer: "BETAFPV", Model: "BETAFPV Moxon Antenna (915MHz)",
		WeightG: 18.0, Connector: "SMA", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "536e9a77-e3d8-4743-93d3-449a473716ee", Id: "betafpv-900mhz-dipole-t-antenna", Manufacturer: "BETAFPV", Model: "BETAFPV 900MHz Dipole T-Antenna",
		WeightG: 0.94, Connector: "U.FL", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "206c4d0e-50cf-4bd3-8536-771a3f26aa0c", Id: "matek-900mhz-receiver-dipole-antenna", Manufacturer: "Matek", Model: "Matek 900MHz Receiver Dipole Antenna",
		WeightG: 1.0, Connector: "U.FL", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "32893f6b-73e0-4caa-9268-11a087924a0a", Id: "truerc-singularity-915-(rhcp)", Manufacturer: "TrueRC", Model: "TrueRC Singularity 915 (RHCP)",
		WeightG: 24.0, Connector: "SMA", Polarization: "RHCP", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "d2d64a3e-f109-4eca-838b-d8b27e2631f7", Id: "truerc-true-mox-915", Manufacturer: "TrueRC", Model: "TrueRC True-MoX 915",
		WeightG: 22.0, Connector: "SMA", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "57381978-90fc-4994-b4a9-af06429fc70d", Id: "truerc-bardpole-915mhz", Manufacturer: "TrueRC", Model: "TrueRC BARDpole 915MHz",
		WeightG: 6.3, Connector: "U.FL", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "7d905b95-a4cb-4e03-aba4-f488f4ecf871", Id: "happymodel-915mhz-micro-t-antenna", Manufacturer: "Happymodel", Model: "Happymodel 915MHz Micro T Antenna",
		WeightG: 0.9, Connector: "U.FL", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
	if err := pb.CreateAntenna(ctx, tx, &pb.Antenna{
		Uuid: "d1add965-ee8e-4623-a4d6-b7ae925694a4", Id: "happymodel-es900tx-stock-transmitter-antenna", Manufacturer: "Happymodel", Model: "Happymodel ES900TX Stock Transmitter Antenna",
		WeightG: 12.0, Connector: "SMA", Polarization: "Linear", FrequencyBandGhz: 5.8,
	}); err != nil { panic(err) }
}
