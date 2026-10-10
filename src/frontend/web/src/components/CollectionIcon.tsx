import type { FC, SVGProps } from "react";
import {
  Antenna,
  Battery,
  Camera,
  Satellite,
  Cpu,
  Radio,
  Tv,
  Wrench,
  Zap,
  Box,
  Hammer,
  type LucideProps,
} from "lucide-react";
import { normalizeCollectionKey } from "../lib/collectionColors";

export type IconComponent = FC<LucideProps & SVGProps<SVGSVGElement>>;

/**
 * Custom FPV 3-Blade Propeller Icon matching Lucide 24x24 icon grid.
 */
export const Propeller: IconComponent = ({
  size = 24,
  className = "",
  strokeWidth = 2,
  ...props
}) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth={strokeWidth}
    strokeLinecap="round"
    strokeLinejoin="round"
    className={className}
    {...props}
  >
    {/* Blade 1 (pointing up) */}
    <path d="M10.8 10.5C9.2 7.8 9.5 4.5 11.5 2.5c.9-.9 2-.7 2.4.4.9 2.2 0 5.2-1.5 7.6" />
    {/* Blade 2 (rotated 120deg) */}
    <path
      d="M10.8 10.5C9.2 7.8 9.5 4.5 11.5 2.5c.9-.9 2-.7 2.4.4.9 2.2 0 5.2-1.5 7.6"
      transform="rotate(120 12 12)"
    />
    {/* Blade 3 (rotated 240deg) */}
    <path
      d="M10.8 10.5C9.2 7.8 9.5 4.5 11.5 2.5c.9-.9 2-.7 2.4.4.9 2.2 0 5.2-1.5 7.6"
      transform="rotate(240 12 12)"
    />
    {/* Center hub and motor shaft */}
    <circle cx="12" cy="12" r="2.5" />
    <circle cx="12" cy="12" r="1" />
  </svg>
);

export const PropellerIcon = Propeller;

/**
 * Custom FPV Drone Frame (True-X quadcopter frame) matching Lucide 24x24 icon grid.
 */
export const DroneFrame: IconComponent = ({
  size = 24,
  className = "",
  strokeWidth = 2,
  ...props
}) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth={strokeWidth}
    strokeLinecap="round"
    strokeLinejoin="round"
    className={className}
    {...props}
  >
    {/* 4 Carbon Arms */}
    <line x1="8.5" y1="8.5" x2="5.5" y2="5.5" />
    <line x1="15.5" y1="8.5" x2="18.5" y2="5.5" />
    <line x1="8.5" y1="15.5" x2="5.5" y2="18.5" />
    <line x1="15.5" y1="15.5" x2="18.5" y2="18.5" />
    {/* 4 Motor Mount Pads */}
    <circle cx="4" cy="4" r="2.5" />
    <circle cx="20" cy="4" r="2.5" />
    <circle cx="4" cy="20" r="2.5" />
    <circle cx="20" cy="20" r="2.5" />
    {/* Center Fuselage / Stack Plates */}
    <rect x="8.5" y="7" width="7" height="10" rx="1.5" />
    <circle cx="12" cy="12" r="1.2" />
  </svg>
);

export const DroneFrameIcon = DroneFrame;

/**
 * Custom FPV Brushless Motor (low-profile pancake outrunner) matching Lucide 24x24 icon grid.
 */
export const Motor: IconComponent = ({ size = 24, className = "", strokeWidth = 2, ...props }) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth={strokeWidth}
    strokeLinecap="round"
    strokeLinejoin="round"
    className={className}
    {...props}
  >
    {/* Prop Shaft with M5 thread notch */}
    <line x1="12" y1="2" x2="12" y2="6.5" />
    <line x1="10.5" y1="4" x2="13.5" y2="4" />
    {/* Low-profile wide bell */}
    <path d="M3.5 11 7 7.5h10l3.5 3.5v4H3.5z" />
    {/* Horizontal bell cooling slit */}
    <line x1="7" y1="11" x2="17" y2="11" />
    {/* Stator core housing & copper windings */}
    <path d="M5 15v3.5h14V15" />
    <line x1="8.5" y1="15" x2="8.5" y2="18.5" />
    <line x1="12" y1="15" x2="12" y2="18.5" />
    <line x1="15.5" y1="15" x2="15.5" y2="18.5" />
    {/* Baseplate */}
    <line x1="2.5" y1="20.5" x2="21.5" y2="20.5" />
  </svg>
);

export const MotorIcon = Motor;
export const BrushlessMotor = Motor;

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
      return DroneFrame;
    case "gps-receivers":
      return Satellite;
    case "motors":
      return Motor;
    case "propellers":
      return Propeller;
    case "receivers":
      return Radio;
    case "video-transmitters":
      return Tv;
    case "the-forge":
    case "forge":
      return Hammer;
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
