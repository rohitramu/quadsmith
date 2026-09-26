import "dotenv/config";
import { PrismaClient } from '@prisma/client';
import { PrismaPg } from '@prisma/adapter-pg';
import pg from 'pg';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const connectionString = process.env.DATABASE_URL;
const pool = new pg.Pool({ connectionString });
const adapter = new PrismaPg(pool);
const prisma = new PrismaClient({ adapter });

async function main() {
  const dataPath = path.join(__dirname, 'data.json');
  const rawData = fs.readFileSync(dataPath, 'utf8');
  const data = JSON.parse(rawData);

  console.log('Clearing database...');
  // Wipe in correct order to respect constraints
  await prisma.build.deleteMany();
  await prisma.receiverConfiguration.deleteMany();
  await prisma.vtxConfiguration.deleteMany();
  await prisma.incompatibilityComponent.deleteMany();
  await prisma.componentCompany.deleteMany();
  await prisma.referenceLink.deleteMany();
  
  await prisma.frameFcStackMount.deleteMany();
  await prisma.frameVtxMount.deleteMany();

  await prisma.antenna.deleteMany();
  await prisma.gps.deleteMany();
  await prisma.receiver.deleteMany();
  await prisma.camera.deleteMany();
  await prisma.vtx.deleteMany();
  await prisma.esc.deleteMany();
  await prisma.flightController.deleteMany();
  await prisma.propeller.deleteMany();
  await prisma.motor.deleteMany();
  await prisma.frame.deleteMany();
  await prisma.battery.deleteMany();
  await prisma.goggles.deleteMany();
  await prisma.transmitter.deleteMany();

  await prisma.hardwareComponent.deleteMany();
  await prisma.softwareComponent.deleteMany();
  await prisma.component.deleteMany();

  await prisma.company.deleteMany();
  await prisma.rfFrequency.deleteMany();
  await prisma.rfProtocol.deleteMany();
  await prisma.antennaPolarization.deleteMany();
  await prisma.antennaConnector.deleteMany();
  await prisma.batteryConnector.deleteMany();
  await prisma.vtxEcosystem.deleteMany();
  await prisma.tag.deleteMany();
  await prisma.motorMountPattern.deleteMany();
  await prisma.boardMountPattern.deleteMany();
  await prisma.batteryChemistry.deleteMany();
  await prisma.incompatibilityIssue.deleteMany();

  console.log('Seeding lookup tables...');
  if (data.companies) {
    for (const c of data.companies) {
      await prisma.company.create({ data: { name: c.name } });
    }
  }
  if (data.rfFrequencies) {
    for (const e of data.rfFrequencies) await prisma.rfFrequency.create({ data: e });
  }
  if (data.rfProtocols) {
    for (const e of data.rfProtocols) await prisma.rfProtocol.create({ data: e });
  }
  if (data.antennaPolarizations) {
    for (const e of data.antennaPolarizations) await prisma.antennaPolarization.create({ data: e });
  }
  if (data.antennaConnectors) {
    for (const e of data.antennaConnectors) await prisma.antennaConnector.create({ data: e });
  }
  if (data.batteryConnectors) {
    for (const e of data.batteryConnectors) await prisma.batteryConnector.create({ data: e });
  }
  if (data.vtxEcosystems) {
    for (const e of data.vtxEcosystems) await prisma.vtxEcosystem.create({ data: e });
  }
  if (data.tags) {
    for (const e of data.tags) await prisma.tag.create({ data: e });
  }
  if (data.motorMountPatterns) {
    for (const e of data.motorMountPatterns) await prisma.motorMountPattern.create({ data: e });
  }
  if (data.boardMountPatterns) {
    for (const e of data.boardMountPatterns) await prisma.boardMountPattern.create({ data: e });
  }
  if (data.batteryChemistries) {
    for (const e of data.batteryChemistries) {
    if (e.id === "LiPo") await prisma.batteryChemistry.create({ data: { name: "LiPo", nominalVoltagePerCellV: 3.7, maxVoltagePerCellV: 4.2, minVoltagePerCellV: 3.2 } });
    if (e.id === "LiHV") await prisma.batteryChemistry.create({ data: { name: "LiHV", nominalVoltagePerCellV: 3.8, maxVoltagePerCellV: 4.35, minVoltagePerCellV: 3.2 } });
    if (e.id === "Li-Ion") await prisma.batteryChemistry.create({ data: { name: "Li-Ion", nominalVoltagePerCellV: 3.6, maxVoltagePerCellV: 4.2, minVoltagePerCellV: 2.5 } });
  }
  }
  if (data.incompatibilityIssues) {
    for (const e of data.incompatibilityIssues) await prisma.incompatibilityIssue.create({ data: e });
  }

  console.log(`Seeding ${data.hardwareComponents.length} hardware components...`);
  
  const componentIdMap = new Map<string, number>();
  
  const allCompanies = await prisma.company.findMany();
  const getCompanyId = (name: string) => {
    const c = allCompanies.find(c => c.name === name);
    return c ? c.id : null;
  };

  
  const allComponents = [...(data.hardwareComponents || []), ...(data.softwareComponents || [])];
  console.log(`Seeding ${allComponents.length} components...`);

  for (const comp of allComponents) {
    const companiesConnect = (comp.referenceLinks || []).map((l: any) => getCompanyId(l.companyName)).filter((id: any) => id !== null);
    const uniqueCompanyIds = [...new Set(companiesConnect)] as number[];

    
    const baseComp = await prisma.component.create({
      data: {
        name: comp.name || comp.modelName,
        releaseYear: comp.releaseYear,
        releaseMonth: comp.releaseMonth,
        releaseDay: comp.releaseDay,
        companies: {
          create: uniqueCompanyIds.map((id) => ({
            company: { connect: { id } }
          }))
        },
        referenceLinks: {
          create: (comp.referenceLinks || []).map((l: any) => ({
            url: l.url,
            linkType: l.linkType,
            title: l.title || l.linkType,
            description: l.description,
            companyId: getCompanyId(l.companyName)
          }))
        }
      }
    });

    componentIdMap.set(comp.name || comp.modelName, baseComp.id);

    
    if (comp.isSoftware) {
      await prisma.softwareComponent.create({
        data: {
          id: baseComp.id,
          version: comp.version
        }
      });
      
      if (comp.softwareType === "Firmware") {
        await prisma.fcFirmware.create({
          data: {
            id: baseComp.id,
            supportsGps: comp.name === "INAV" || comp.name === "Betaflight",
            isOpenSource: comp.name !== "Walksnail Avatar OS"
          }
        });
      } else if (comp.softwareType === "ESC Firmware") {
        await prisma.escFirmware.create({
          data: {
            id: baseComp.id,
            supportsBidirectionalDshot: true
          }
        });
      } else if (comp.softwareType === "Operating System") {
        await prisma.operatingSystem.create({
          data: {
            id: baseComp.id,
            supportsLuaScripts: comp.name === "EdgeTX"
          }
        });
      } else if (comp.softwareType === "Configurator") {
        await prisma.configurator.create({
          data: {
            id: baseComp.id,
            hasMobileApp: comp.name === "Betaflight Configurator" || comp.name === "ExpressLRS Configurator",
            hasWebApp: comp.name === "ExpressLRS Configurator"
          }
        });
      }

      continue;
    }

    const hw = await prisma.hardwareComponent.create({
      data: {
        id: baseComp.id,
        weightG: comp.weightG
      }
    });


    

    if (comp.frame) {
      await prisma.frame.create({
        data: {
          id: baseComp.id,
          wheelbaseMm: comp.frame.wheelbaseMm,
          
          
          
          
          maxPropSizeMm: comp.frame.maxPropSizeMm || 5,
          cameraMountWidthMm: comp.frame.cameraMountWidthMm || 19,
          motorMountPatterns: { connect: (comp.frame.motorMounts || []).map((id: string) => ({ id })) },
          fcStackMounts: {
             create: (comp.frame.fcStackMounts || []).map((id: string) => ({ boardMountPatternId: id }))
          },
          vtxMounts: {
             create: (comp.frame.vtxMounts || []).map((id: string) => ({ boardMountPatternId: id }))
          }
        }
      });
    }

    if (comp.motor) {
      await prisma.motor.create({
        data: {
          id: baseComp.id,
          statorSize: comp.motor.statorSize || "Unknown",
          kvRating: comp.motor.kv || 0,
          inputVoltageMinV: comp.motor.inputVoltageMinV || 0,
          inputVoltageMaxV: comp.motor.inputVoltageMaxV || 0,
          mountPatterns: { connect: (comp.motor.mountPatterns || []).map((id: string) => ({ id })) },
          mountBoltSize: comp.motor.mountBoltSize || "M2",
          shaftType: comp.motor.shaftType || "Unknown",
          maxCurrentA: comp.motor.maxCurrentA
        }
      });
    }

    if (comp.propeller) {
      await prisma.propeller.create({
        data: {
          id: baseComp.id,
          diameterMm: comp.propeller.diameterMm,
          pitchMm: comp.propeller.pitchMm,
          bladeCount: comp.propeller.bladeCount,
          mountType: comp.propeller.mountType || "M5"
        }
      });
    }

    if (comp.flightController) {
      await prisma.flightController.create({
        data: {
          id: baseComp.id,
          mcuProcessor: comp.flightController.mcuProcessor || "Unknown",
          gyroSensor: comp.flightController.gyroSensor,
          mountPatterns: { connect: (comp.flightController.mountPatterns || []).map((id: string) => ({ id })) },
          boardHeightMm: comp.flightController.boardHeightMm || 0,
          inputVoltageMinV: comp.flightController.inputVoltageMinV || 0,
          inputVoltageMaxV: comp.flightController.inputVoltageMaxV || 0,
          supportedFirmware: comp.flightController.supportedFirmware || [],
          escInterface: comp.flightController.escInterface || "Unknown"
        }
      });
    }
    
    if (comp.esc) {
       await prisma.esc.create({
         data: {
           id: baseComp.id,
           continuousCurrentA: comp.esc.continuousCurrentA || 0,
           burstCurrentA: comp.esc.burstCurrentA || 0,
           firmwareProtocol: comp.esc.firmwareProtocol || "Unknown",
           isIntegrated: comp.esc.isIntegrated || false,
           formFactor: comp.esc.formFactor || "4-in-1",
           mountPatterns: { connect: (comp.esc.mountPatterns || []).map((id: string) => ({ id })) },
           boardHeightMm: comp.esc.boardHeightMm || 0,
           inputVoltageMinV: comp.esc.inputVoltageMinV || 0,
           inputVoltageMaxV: comp.esc.inputVoltageMaxV || 0
         }
       });
    }

    if (comp.vtx) {
       await prisma.vtx.create({
         data: {
           id: baseComp.id,
           ecosystemId: comp.vtx.ecosystemId,
           videoConnectionStandard: comp.vtx.videoConnectionStandard,
           boardHeightMm: comp.vtx.boardHeightMm || 0,
           inputVoltageMinV: comp.vtx.inputVoltageMinV || 0,
           inputVoltageMaxV: comp.vtx.inputVoltageMaxV || 0,
           maxPowerMw: comp.vtx.maxPowerMw,
           antennaCount: comp.vtx.antennaCount || 1,
           antennaConnectorId: comp.vtx.antennaConnectorId,
           includedAntennaPolarizationId: comp.vtx.includedAntennaPolarizationId,
           mountPatterns: { connect: (comp.vtx.mountPatterns || []).map((id: string) => ({ id })) },
           supportedFrequencies: { connect: (comp.vtx.supportedFrequencies || []).map((id: string) => ({ id })) }
         }
       });
    }
    
    if (comp.camera) {
       await prisma.camera.create({
         data: {
           id: baseComp.id,
           ecosystemId: comp.camera.ecosystemId,
           videoConnectionStandard: comp.camera.videoConnectionStandard,
           widthMm: comp.camera.widthMm || 14,
           heightMm: comp.camera.heightMm || 14,
           depthMm: comp.camera.depthMm || 14,
           inputVoltageMinV: comp.camera.inputVoltageMinV || 0,
           inputVoltageMaxV: comp.camera.inputVoltageMaxV || 0,
           mountingScrewSize: comp.camera.mountingScrewSize || "M2",
           aspectRatio: comp.camera.aspectRatio || "16:9"
         }
       });
    }
    
    if (comp.receiver) {
       await prisma.receiver.create({
         data: {
           id: baseComp.id,
           outputProtocol: comp.receiver.outputProtocol || "Unknown",
           inputVoltageMinV: comp.receiver.inputVoltageMinV || 0,
           inputVoltageMaxV: comp.receiver.inputVoltageMaxV || 0,
           antennaCount: comp.receiver.antennaCount || 1,
           antennaConnectorId: comp.receiver.antennaConnectorId,
           includedAntennaPolarizationId: comp.receiver.includedAntennaPolarizationId,
           rfProtocols: { connectOrCreate: (comp.receiver.outputProtocol ? [{ where: { id: comp.receiver.outputProtocol }, create: { id: comp.receiver.outputProtocol } }] : []) },
           supportedFrequencies: { connect: (comp.receiver.supportedFrequencies || []).map((id: string) => ({ id })) }
         }
       });
    }

    if (comp.gps) {
       await prisma.gps.create({
         data: {
           id: baseComp.id,
           chipset: comp.gps.chipset || "Unknown",
           hasCompass: comp.gps.hasCompass || false,
           inputVoltageMinV: comp.gps.inputVoltageMinV || 0,
           inputVoltageMaxV: comp.gps.inputVoltageMaxV || 0,
           lengthMm: comp.gps.lengthMm,
           widthMm: comp.gps.widthMm,
           heightMm: comp.gps.heightMm
         }
       });
    }

    if (comp.antenna) {
       await prisma.antenna.create({
         data: {
           id: baseComp.id,
           supportedFrequencyId: comp.antenna.supportedFrequencyId || "5.8GHz",
           polarizationId: comp.antenna.polarizationId || "RHCP",
           connectorId: comp.antenna.connectorId || "SMA",
           antennaStyle: comp.antenna.antennaStyle || "Omni",
           gainDbi: comp.antenna.gainDbi,
           cableLengthMm: comp.antenna.cableLengthMm
         }
       });
    }

    if (comp.battery) {
       const chem = await prisma.batteryChemistry.findFirst({ where: { name: "LiPo" } });
       await prisma.battery.create({
         data: {
           id: baseComp.id,
           capacityMah: comp.battery.capacityMah,
           cellCountS: comp.battery.cellCountS || comp.battery.cellCount || 1,
           cellCountP: comp.battery.cellCountP || 1,
           continuousCRating: comp.battery.continuousCRating || comp.battery.cRating || 0,
           
           connectorTypeId: comp.battery.connectorTypeId || "XT60",
             chemistryId: chem!.id,
           lengthMm: comp.battery.lengthMm,
           widthMm: comp.battery.widthMm,
           heightMm: comp.battery.heightMm
         }
       });
    }
    
    if (comp.goggles) {
      await prisma.goggles.create({
        data: {
          id: baseComp.id,
          ecosystemId: comp.goggles.ecosystemId || "Analog",
          antennaCount: comp.goggles.antennaCount || 2,
          antennaConnectorId: comp.goggles.antennaConnectorId,
          includedAntennaPolarizationId: comp.goggles.includedAntennaPolarizationId
        }
      });
    }
    
    if (comp.transmitter) {
      await prisma.transmitter.create({
        data: {
          id: baseComp.id,
          externalBayType: comp.transmitter.externalBayType || "Nano",
          operatingSystem: comp.transmitter.operatingSystem || "EdgeTX",
          antennaCount: comp.transmitter.antennaCount || 1,
          antennaConnectorId: comp.transmitter.antennaConnectorId,
          rfProtocols: { connect: (comp.transmitter.protocol ? [{ id: comp.transmitter.protocol }] : []) }
        }
      });
    }
  }

  console.log(`Seeding builds...`);
  if (data.builds) {
    for (const b of data.builds) {
      if (!b.components) continue;
      
      const vtxId = componentIdMap.get(b.components.vtxConfiguration?.vtxName);
      let vtxConfigId = null;
      if (vtxId) {
         const antennasConnect = (b.components.vtxConfiguration.antennaNames || []).map((name: string) => componentIdMap.get(name)).filter(Boolean);
         const created = await prisma.vtxConfiguration.create({
           data: {
             vtxId: vtxId,
             loadoutName: b.components.vtxConfiguration.loadoutName || "Standard",
             antennas: { connect: antennasConnect.map((id: number) => ({ id })) }
           }
         });
         vtxConfigId = created.id;
      }

      const receiverId = componentIdMap.get(b.components.receiverConfiguration?.receiverName);
      let receiverConfigId = null;
      if (receiverId) {
         const antennasConnect = (b.components.receiverConfiguration.antennaNames || []).map((name: string) => componentIdMap.get(name)).filter(Boolean);
         const created = await prisma.receiverConfiguration.create({
           data: {
             receiverId: receiverId,
             loadoutName: b.components.receiverConfiguration.loadoutName || "Standard",
             antennas: { connect: antennasConnect.map((id: number) => ({ id })) }
           }
         });
         receiverConfigId = created.id;
      }

      
      console.log('Build components:', b.components);
      console.log('Frame ID:', componentIdMap.get(b.components.frameName));
      console.log('Motor ID:', componentIdMap.get(b.components.motorName));
      console.log('Prop ID:', componentIdMap.get(b.components.propellerName));
      console.log('FC ID:', componentIdMap.get(b.components.flightControllerName));
      console.log('Camera ID:', componentIdMap.get(b.components.cameraName));
      
      const connectIf = (id: any): any => id ? { connect: { id } } : undefined;
      
      await prisma.build.create({
        data: {
          name: b.name,
          crashResistanceRating: b.crashResistanceRating,
          description: b.description,
          isVerified: b.isVerified,
          miscWeightG: b.miscWeightG,
          tags: { connectOrCreate: (b.tags || []).map((t: string) => ({ where: { id: t }, create: { id: t } })) },
          
          frame: connectIf(componentIdMap.get(b.components.frameName)),
          motor: connectIf(componentIdMap.get(b.components.motorName)),
          propeller: connectIf(componentIdMap.get(b.components.propellerName)),
          flightController: connectIf(componentIdMap.get(b.components.flightControllerName)),
          esc: connectIf(componentIdMap.get(b.components.escName)),
          camera: connectIf(componentIdMap.get(b.components.cameraName)),
          gps: connectIf(componentIdMap.get(b.components.gpsName)),
          
          vtxConfig: connectIf(vtxConfigId),
          receiverConfig: connectIf(receiverConfigId),
        }
      });
    }
  }

  console.log('Database seeded successfully!');
}

main()
  .catch(e => {
    console.error(e);
    process.exit(1);
  })
  .finally(async () => {
    await prisma.$disconnect();
    await pool.end();
  });
