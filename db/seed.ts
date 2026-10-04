import { PrismaClient, ComponentType } from '@prisma/client';
import { Pool } from 'pg';
import { PrismaPg } from '@prisma/adapter-pg';
import dotenv from 'dotenv';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

dotenv.config();

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const connectionString = process.env.DATABASE_URL;
const pool = new Pool({ connectionString });
const adapter = new PrismaPg(pool);
const prisma = new PrismaClient({ adapter });

async function main() {
  const dataPath = path.join(__dirname, 'data.json');
  const rawData = fs.readFileSync(dataPath, 'utf8');
  const data = JSON.parse(rawData);

  console.log('Clearing database...');
  await prisma.resource.deleteMany();

  console.log('Seeding lookup tables...');
  
  // Maps to store name -> UUIDv7 String ID
  const mapCompany = new Map<string, string>();
  if (data.companies) {
    for (const c of data.companies) {
      const res = await prisma.resource.create({
        data: {
          resourceType: 'COMPANY',
          company: { create: { name: c.name, websiteUrl: c.websiteUrl } }
        },
        include: { company: true }
      });
      mapCompany.set(c.name, res.company!.id);
      if (c.id) mapCompany.set(c.id, res.company!.id);
    }
  }

  const mapFreq = new Map<string, string>();
  if (data.rfFrequencies) {
    for (const e of data.rfFrequencies) {
      const res = await prisma.resource.create({
        data: {
          resourceType: 'RF_FREQUENCY',
          rfFrequency: { create: { name: e.id } }
        },
        include: { rfFrequency: true }
      });
      mapFreq.set(e.id, res.rfFrequency!.id);
    }
  }

  const mapProtocol = new Map<string, string>();
  if (data.rfProtocols) {
    for (const e of data.rfProtocols) {
      const res = await prisma.resource.create({
        data: {
          resourceType: 'RF_PROTOCOL',
          rfProtocol: { create: { name: e.id } }
        },
        include: { rfProtocol: true }
      });
      mapProtocol.set(e.id, res.rfProtocol!.id);
    }
  }

  const mapPol = new Map<string, string>();
  if (data.antennaPolarizations) {
    for (const e of data.antennaPolarizations) {
      const res = await prisma.resource.create({
        data: {
          resourceType: 'ANTENNA_POLARIZATION',
          antennaPolarization: { create: { name: e.id } }
        },
        include: { antennaPolarization: true }
      });
      mapPol.set(e.id, res.antennaPolarization!.id);
    }
  }

  const mapConn = new Map<string, string>();
  if (data.antennaConnectors) {
    for (const e of data.antennaConnectors) {
      const res = await prisma.resource.create({
        data: {
          resourceType: 'ANTENNA_CONNECTOR',
          antennaConnector: { create: { name: e.id } }
        },
        include: { antennaConnector: true }
      });
      mapConn.set(e.id, res.antennaConnector!.id);
    }
  }

  const mapBattConn = new Map<string, string>();
  if (data.batteryConnectors) {
    for (const e of data.batteryConnectors) {
      const res = await prisma.resource.create({
        data: {
          resourceType: 'BATTERY_CONNECTOR',
          batteryConnector: { create: { name: e.id, maxCurrentA: e.maxCurrentA, maxVoltageV: e.maxVoltageV } }
        },
        include: { batteryConnector: true }
      });
      mapBattConn.set(e.id, res.batteryConnector!.id);
    }
  }

  const mapEcosystem = new Map<string, string>();
  if (data.vtxEcosystems) {
    for (const e of data.vtxEcosystems) {
      const res = await prisma.resource.create({
        data: {
          resourceType: 'VTX_ECOSYSTEM',
          vtxEcosystem: { create: { name: e.id } }
        },
        include: { vtxEcosystem: true }
      });
      mapEcosystem.set(e.id, res.vtxEcosystem!.id);
    }
  }

  const mapTag = new Map<string, string>();
  if (data.tags) {
    for (const e of data.tags) {
      const res = await prisma.resource.create({
        data: {
          resourceType: 'TAG',
          tag: { create: { name: e.id, description: e.description } }
        },
        include: { tag: true }
      });
      mapTag.set(e.id, res.tag!.id);
    }
  }

  const mapMotorMount = new Map<string, string>();
  if (data.motorMountPatterns) {
    for (const e of data.motorMountPatterns) {
      const res = await prisma.resource.create({
        data: {
          resourceType: 'MOTOR_MOUNT_PATTERN',
          motorMountPattern: { create: { name: e.id } }
        },
        include: { motorMountPattern: true }
      });
      mapMotorMount.set(e.id, res.motorMountPattern!.id);
    }
  }

  const mapBoardMount = new Map<string, string>();
  if (data.boardMountPatterns) {
    for (const e of data.boardMountPatterns) {
      const res = await prisma.resource.create({
        data: {
          resourceType: 'BOARD_MOUNT_PATTERN',
          boardMountPattern: { create: { name: e.id } }
        },
        include: { boardMountPattern: true }
      });
      mapBoardMount.set(e.id, res.boardMountPattern!.id);
    }
  }

  const mapBattChem = new Map<string, string>();
  if (data.batteryChemistries) {
    for (const e of data.batteryChemistries) {
      const name = e.name || e.id;
      const res = await prisma.resource.create({
        data: {
          resourceType: 'BATTERY_CHEMISTRY',
          batteryChemistry: {
            create: {
              name: name,
              nominalVoltagePerCellV: e.nominalVoltagePerCellV || 3.7,
              maxVoltagePerCellV: e.maxVoltagePerCellV || 4.2,
              minVoltagePerCellV: e.minVoltagePerCellV || 3.2
            }
          }
        },
        include: { batteryChemistry: true }
      });
      mapBattChem.set(name, res.batteryChemistry!.id);
      mapBattChem.set(e.id, res.batteryChemistry!.id);
    }
  }

  console.log('Seeding Hardware (Protobuf JSONB)...');
  const compMap = new Map<string, string>();

  if (data.hardwareComponents) {
    for (const comp of data.hardwareComponents) {
      let compType: ComponentType = ComponentType.FRAME;
      const profileData: Record<string, any> = {};

      if (comp.weightG != null) {
        profileData.weightG = comp.weightG;
      }

      if (comp.frame) {
        compType = ComponentType.FRAME;
        profileData.frame = {
          wheelbaseMm: comp.frame.wheelbaseMm || 0,
          maxPropSizeMm: comp.frame.maxPropSizeMm || 0,
          cameraMountWidthMm: comp.frame.cameraMountWidthMm || 0,
          maxBatteryLengthMm: comp.frame.maxBatteryLengthMm,
          maxBatteryWidthMm: comp.frame.maxBatteryWidthMm,
          maxBatteryHeightMm: comp.frame.maxBatteryHeightMm,
          fcStackMountIds: (comp.frame.fcStackMounts || []).map((m: any) => mapBoardMount.get(m.id)).filter(Boolean),
          vtxMountIds: (comp.frame.vtxMounts || []).map((m: any) => mapBoardMount.get(m.id)).filter(Boolean),
          motorMountPatternIds: (comp.frame.motorMountPatterns || []).map((m: any) => mapMotorMount.get(m.id)).filter(Boolean)
        };
      } else if (comp.motor) {
        compType = ComponentType.MOTOR;
        profileData.motor = {
          statorSize: comp.motor.statorSize || "Unknown",
          kvRating: comp.motor.kvRating || 0,
          inputVoltageMinV: comp.motor.inputVoltageMinV || 0,
          inputVoltageMaxV: comp.motor.inputVoltageMaxV || 0,
          mountBoltSize: comp.motor.mountBoltSize || "Unknown",
          shaftType: comp.motor.shaftType || "Unknown",
          maxCurrentA: comp.motor.maxCurrentA,
          mountPatternIds: (comp.motor.mountPatterns || []).map((m: any) => mapMotorMount.get(m.id)).filter(Boolean)
        };
      } else if (comp.propeller) {
        compType = ComponentType.PROPELLER;
        profileData.propeller = {
          diameterMm: comp.propeller.diameterMm || 0,
          pitchMm: comp.propeller.pitchMm || 0,
          bladeCount: comp.propeller.bladeCount || 0,
          mountType: comp.propeller.mountType || "Unknown",
          recommendedStators: comp.propeller.recommendedStator || []
        };
      } else if (comp.flightController) {
        compType = ComponentType.FLIGHT_CONTROLLER;
        profileData.flightController = {
          boardHeightMm: comp.flightController.boardHeightMm || 0,
          mcuProcessor: comp.flightController.mcuProcessor || "Unknown",
          gyroSensor: comp.flightController.gyroSensor,
          inputVoltageMinV: comp.flightController.inputVoltageMinV || 0,
          inputVoltageMaxV: comp.flightController.inputVoltageMaxV || 0,
          escInterface: comp.flightController.escInterface || "DShot300",
          mountPatternIds: (comp.flightController.mountPatterns || []).map((m: any) => mapBoardMount.get(m.id)).filter(Boolean),
          becOutputsJson: JSON.stringify(comp.flightController.becOutputs || {}),
          uartConnectionsJson: JSON.stringify(comp.flightController.uartConnections || {})
        };
      } else if (comp.esc) {
        compType = ComponentType.ESC;
        let ff = 'FOUR_IN_ONE';
        if (comp.esc.formFactor === 'Individual') ff = 'INDIVIDUAL';
        if (comp.esc.formFactor === 'INTEGRATED_IN_FC') ff = 'INTEGRATED_IN_FC';

        profileData.esc = {
          formFactor: ff,
          boardHeightMm: comp.esc.boardHeightMm || 0,
          continuousCurrentA: comp.esc.continuousCurrentA || 0,
          burstCurrentA: comp.esc.burstCurrentA || 0,
          inputVoltageMinV: comp.esc.inputVoltageMinV || 0,
          inputVoltageMaxV: comp.esc.inputVoltageMaxV || 0,
          mountPatternIds: (comp.esc.mountPatterns || []).map((m: any) => mapBoardMount.get(m.id)).filter(Boolean)
        };
      } else if (comp.battery) {
        compType = ComponentType.BATTERY;
        profileData.battery = {
          chemistryId: mapBattChem.get(comp.battery.chemistryId) || "",
          cellCountS: comp.battery.cellCountS || 0,
          cellCountP: comp.battery.cellCountP || 1,
          capacityMah: comp.battery.capacityMah || 0,
          continuousCRating: comp.battery.continuousCRating || 0,
          connectorTypeId: mapBattConn.get(comp.battery.connectorTypeId) || "",
          lengthMm: comp.battery.lengthMm,
          widthMm: comp.battery.widthMm,
          heightMm: comp.battery.heightMm
        };
      } else if (comp.vtx) {
        compType = ComponentType.VTX;
        profileData.vtx = {
          ecosystemId: mapEcosystem.get(comp.vtx.ecosystemId) || "",
          videoConnectionStandard: comp.vtx.videoConnectionStandard,
          boardHeightMm: comp.vtx.boardHeightMm || 0,
          inputVoltageMinV: comp.vtx.inputVoltageMinV || 0,
          inputVoltageMaxV: comp.vtx.inputVoltageMaxV || 0,
          maxPowerMw: comp.vtx.maxPowerMw,
          antennaCount: comp.vtx.antennaCount || 1,
          antennaConnectorId: mapConn.get(comp.vtx.antennaConnectorId),
          includedAntennaPolarizationId: mapPol.get(comp.vtx.includedAntennaPolarizationId),
          mountPatternIds: (comp.vtx.mountPatterns || []).map((m: any) => mapBoardMount.get(m.id)).filter(Boolean),
          supportedFrequencyIds: (comp.vtx.supportedFrequencies || []).map((f: any) => mapFreq.get(f.id)).filter(Boolean)
        };
      } else if (comp.camera) {
        compType = ComponentType.CAMERA;
        profileData.camera = {
          ecosystemId: mapEcosystem.get(comp.camera.ecosystemId) || "",
          videoConnectionStandard: comp.camera.videoConnectionStandard,
          widthMm: comp.camera.widthMm || 0,
          heightMm: comp.camera.heightMm || 0,
          depthMm: comp.camera.depthMm || 0,
          inputVoltageMinV: comp.camera.inputVoltageMinV || 0,
          inputVoltageMaxV: comp.camera.inputVoltageMaxV || 0,
          mountingScrewSize: comp.camera.mountingScrewSize || "M2",
          aspectRatio: comp.camera.aspectRatio || "16:9"
        };
      } else if (comp.receiver) {
        compType = ComponentType.RECEIVER;
        profileData.receiver = {
          outputProtocol: comp.receiver.outputProtocol || "CRSF",
          inputVoltageMinV: comp.receiver.inputVoltageMinV || 0,
          inputVoltageMaxV: comp.receiver.inputVoltageMaxV || 0,
          antennaCount: comp.receiver.antennaCount || 1,
          antennaConnectorId: mapConn.get(comp.receiver.antennaConnectorId),
          includedAntennaPolarizationId: mapPol.get(comp.receiver.includedAntennaPolarizationId),
          supportedFrequencyIds: (comp.receiver.supportedFrequencies || []).map((f: any) => mapFreq.get(f.id)).filter(Boolean),
          rfProtocolIds: (comp.receiver.rfProtocols || []).map((p: any) => mapProtocol.get(p.id)).filter(Boolean)
        };
      } else if (comp.gps) {
        compType = ComponentType.GPS;
        profileData.gps = {
          chipset: comp.gps.chipset || "Unknown",
          hasCompass: !!comp.gps.hasCompass,
          inputVoltageMinV: comp.gps.inputVoltageMinV || 0,
          inputVoltageMaxV: comp.gps.inputVoltageMaxV || 0,
          lengthMm: comp.gps.lengthMm,
          widthMm: comp.gps.widthMm,
          heightMm: comp.gps.heightMm
        };
      } else if (comp.antenna) {
        compType = ComponentType.ANTENNA;
        profileData.antenna = {
          supportedFrequencyId: mapFreq.get(comp.antenna.supportedFrequencyId) || "",
          polarizationId: mapPol.get(comp.antenna.polarizationId) || "",
          connectorId: mapConn.get(comp.antenna.connectorId) || "",
          antennaStyle: comp.antenna.antennaStyle || "Linear",
          gainDbi: comp.antenna.gainDbi,
          cableLengthMm: comp.antenna.cableLengthMm
        };
      } else if (comp.transmitter) {
        compType = ComponentType.TRANSMITTER;
        profileData.transmitter = {
          externalBayType: comp.transmitter.externalBayType || "None",
          antennaCount: comp.transmitter.antennaCount || 1,
          antennaConnectorId: mapConn.get(comp.transmitter.antennaConnectorId),
          includedAntennaPolarizationId: mapPol.get(comp.transmitter.includedAntennaPolarizationId),
          supportedFrequencyIds: (comp.transmitter.supportedFrequencies || []).map((f: any) => mapFreq.get(f.id)).filter(Boolean),
          rfProtocolIds: (comp.transmitter.rfProtocols || []).map((p: any) => mapProtocol.get(p.id)).filter(Boolean)
        };
      } else if (comp.goggles) {
        compType = ComponentType.GOGGLES;
        profileData.goggles = {
          ecosystemId: mapEcosystem.get(comp.goggles.ecosystemId) || "",
          antennaCount: comp.goggles.antennaCount || 2,
          antennaConnectorId: mapConn.get(comp.goggles.antennaConnectorId),
          includedAntennaPolarizationId: mapPol.get(comp.goggles.includedAntennaPolarizationId),
          supportedFrequencyIds: (comp.goggles.supportedFrequencies || []).map((f: any) => mapFreq.get(f.id)).filter(Boolean)
        };
      }

      const res = await prisma.resource.create({
        data: {
          resourceType: 'COMPONENT',
          component: {
            create: {
              name: comp.modelName,
              type: compType,
              releaseYear: comp.releaseYear,
              data: profileData,
            }
          }
        },
        include: { component: true }
      });

      const compId = res.component!.id;
      compMap.set(comp.modelName, compId);

      if (comp.companies) {
        for (const c of comp.companies) {
          const cid = mapCompany.get(c.companyName);
          if (cid) {
            await prisma.componentCompany.create({
              data: { componentId: compId, companyId: cid }
            });
          }
        }
      }
    }
  }

  console.log('Seeding Software Families...');
  const softwareFamilyMap = new Map<string, string>();
  if (data.softwareFamilies) {
    for (const sf of data.softwareFamilies) {
      const res = await prisma.resource.create({
        data: {
          resourceType: 'SOFTWARE_FAMILY',
          softwareFamily: { create: { name: sf.name } }
        },
        include: { softwareFamily: true }
      });
      softwareFamilyMap.set(sf.name, res.softwareFamily!.id);
    }
  }

  console.log('Seeding Builds & Sub-Assemblies...');
  if (data.builds) {
    for (const b of data.builds) {
      const frameId = compMap.get(b.components.frameName);
      const motorId = compMap.get(b.components.motorName);
      const propId = compMap.get(b.components.propellerName);
      const camId = compMap.get(b.components.cameraName);
      const gpsId = b.components.gpsName ? compMap.get(b.components.gpsName) : undefined;
      const fcId = compMap.get(b.components.flightStack?.flightControllerName);
      const escId = compMap.get(b.components.flightStack?.escName);

      if (!frameId || !motorId || !propId || !camId || !fcId || !escId) {
        continue;
      }

      const stack = await prisma.flightStack.create({
        data: {
          flightControllerId: fcId,
          escId: escId,
          loadoutName: b.components.flightStack.loadoutName || 'Default Stack'
        }
      });

      let vtxConfigId: string | undefined = undefined;
      if (b.components.vtxConfiguration) {
        const vtxId = compMap.get(b.components.vtxConfiguration.vtxName);
        if (vtxId) {
          const antennas = (b.components.vtxConfiguration.antennaNames || [])
            .map((n: string) => compMap.get(n))
            .filter(Boolean) as string[];

          const vc = await prisma.vtxConfiguration.create({
            data: {
              vtxId: vtxId,
              loadoutName: b.components.vtxConfiguration.loadoutName || 'Default VTX Config',
              antennas: { connect: antennas.map(id => ({ id })) }
            }
          });
          vtxConfigId = vc.id;
        }
      }

      let rxConfigId: string | undefined = undefined;
      if (b.components.receiverConfiguration) {
        const rxId = compMap.get(b.components.receiverConfiguration.receiverName);
        if (rxId) {
          const antennas = (b.components.receiverConfiguration.antennaNames || [])
            .map((n: string) => compMap.get(n))
            .filter(Boolean) as string[];

          const rc = await prisma.receiverConfiguration.create({
            data: {
              receiverId: rxId,
              loadoutName: b.components.receiverConfiguration.loadoutName || 'Default RX Config',
              antennas: { connect: antennas.map(id => ({ id })) }
            }
          });
          rxConfigId = rc.id;
        }
      }

      const tags = (b.tags || [])
        .map((t: string) => mapTag.get(t))
        .filter(Boolean) as string[];

      await prisma.resource.create({
        data: {
          resourceType: 'BUILD',
          build: {
            create: {
              name: b.name,
              description: b.description,
              isVerified: b.isVerified ?? false,
              crashResistanceRating: b.crashResistanceRating,
              miscWeightG: b.miscWeightG,
              frameId: frameId,
              motorId: motorId,
              propellerId: propId,
              cameraId: camId,
              gpsId: gpsId,
              flightStackId: stack.id,
              vtxConfigId: vtxConfigId,
              receiverConfigId: rxConfigId,
              tags: { connect: tags.map(id => ({ id })) }
            }
          }
        }
      });
    }
  }

  console.log('Database seeded successfully with Postgres + JSONB + Protobuf architecture!');
}

main()
  .catch((e) => {
    console.error(e);
    process.exit(1);
  })
  .finally(async () => {
    await prisma.$disconnect();
    await pool.end();
  });
