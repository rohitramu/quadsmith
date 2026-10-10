import type { HardwareCollectionDef } from "./hardwareCollections";
import { BuildSchema, type Build } from "../gen/quadsmith/build_pb";
import { listBuilds, getBuild } from "../gen/quadsmith/build-BuildService_connectquery";
import { getCollectionColorFromSchema } from "./collectionColors";

export const BUILD_COLLECTION: HardwareCollectionDef = {
  id: "builds",
  path: "builds",
  name: "Builds",
  description:
    "Community quadcopter builds, curated templates, and custom configurations with live telemetry and compatibility checks.",
  schema: BuildSchema,
  listQuery: listBuilds,
  getQuery: getBuild,
  getDataList: (res: any) => res?.builds ?? [],
  fields: [
    {
      name: "id",
      type: "string",
      description: "Unique human-readable build ID",
      examples: ['id.contains("freestyle")', 'id.contains("5-inch")'],
    },
    {
      name: "uuid",
      type: "string",
      description: "Unique build UUID",
      examples: ['uuid.startsWith("018f")'],
    },
    {
      name: "name",
      type: "string",
      description: "Build title or name",
      examples: ['name.contains("Freestyle")', 'name.contains("7 inch")'],
    },
    {
      name: "description",
      type: "string",
      description: "Build description",
      examples: ['description.contains("racing")', 'description.contains("long range")'],
    },
  ],
  presets: [
    { label: '5" Freestyle', query: 'name.contains("5") || description.contains("5")' },
    { label: '7" Long Range', query: 'name.contains("7") || description.contains("7")' },
    {
      label: "Toothpick",
      query: 'name.contains("Toothpick") || description.contains("Toothpick")',
    },
    {
      label: "Cinewhoop",
      query: 'name.contains("Cinewhoop") || description.contains("Cinewhoop")',
    },
  ],
  columns: {
    name: {
      id: "name",
      title: "Name",
      renderCell: (b: Build) => (
        <span className="font-medium text-zinc-900 dark:text-zinc-100">{b.name || b.id}</span>
      ),
    },
    id: {
      id: "id",
      title: "ID",
      renderCell: (b: Build) => (
        <span className="font-mono text-xs text-zinc-700 dark:text-zinc-300 select-all">
          {b.id}
        </span>
      ),
    },
    uuid: {
      id: "uuid",
      title: "UUID",
      renderCell: (b: Build) => (
        <span className="font-mono text-[11px] text-zinc-500 dark:text-zinc-400 select-all block truncate max-w-[140px]">
          {b.uuid}
        </span>
      ),
    },
    description: {
      id: "description",
      title: "Description",
      renderCell: (b: Build) => (
        <span className="text-xs text-zinc-500 max-w-md truncate block">
          {b.description || "-"}
        </span>
      ),
    },
  },
  defaultColumnIds: ["name", "id", "description"],
  highlights: [
    { label: "Name", value: (b: Build) => b.name },
    { label: "Description", value: (b: Build) => b.description || "-" },
  ],
  technicalSpecs: [
    { label: "ID", value: (b: Build) => b.id },
    { label: "UUID", value: (b: Build) => b.uuid },
  ],
};

BUILD_COLLECTION.color = getCollectionColorFromSchema(BuildSchema, "builds", "Builds");
