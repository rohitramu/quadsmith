import type { FC, SVGProps } from "react";
import {
  Antenna,
  Battery,
  Camera,
  Compass,
  Cpu,
  Layers,
  Radio,
  Rocket,
  Tv,
  Wind,
  Wrench,
  Zap,
  Box,
  type LucideProps,
} from "lucide-react";
import { normalizeCollectionKey } from "../lib/collectionColors";

export type IconComponent = FC<LucideProps & SVGProps<SVGSVGElement>>;

export function getCollectionIconComponent(collectionKey?: string | null): IconComponent {
  const norm = normalizeCollectionKey(collectionKey);
  switch (norm) {
    case "antennas":
      return Antenna;
    case "batteries":
      return Battery;
    case "builds":
      return Wrench;
    case "cameras":
      return Camera;
    case "electronic-speed-controllers":
      return Zap;
    case "flight-controllers":
      return Cpu;
    case "frames":
      return Layers;
    case "gps-receivers":
      return Compass;
    case "motors":
      return Rocket;
    case "propellers":
      return Wind;
    case "receivers":
      return Radio;
    case "video-transmitters":
      return Tv;
    default:
      return Box;
  }
}

export interface CollectionIconProps {
  collection?: string | null;
  size?: number;
  className?: string;
}

export function CollectionIcon({ collection, size = 16, className = "" }: CollectionIconProps) {
  const Component = getCollectionIconComponent(collection);
  return <Component size={size} className={className} aria-hidden="true" />;
}
