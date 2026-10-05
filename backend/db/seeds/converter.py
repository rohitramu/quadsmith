import json
import uuid

def convert():
    with open('data.json', 'r') as f:
        data = json.load(f)
    
    with open('components.textproto', 'w') as out:
        for comp in data.get('hardwareComponents', []):
            out.write('components {\n')
            
            # Use provided ID or generate a new one
            comp_id = comp.get('id', str(uuid.uuid4()))
            out.write(f'  id: "{comp_id}"\n')
            
            name = comp.get('modelName', 'Unknown')
            # Escape quotes
            name = name.replace('"', '\\"')
            out.write(f'  name: "{name}"\n')
            
            if 'releaseYear' in comp:
                out.write(f'  release_year: {comp["releaseYear"]}\n')
            if 'weightG' in comp:
                out.write(f'  weight_g: {comp["weightG"]}\n')
                
            if 'vtx' in comp:
                out.write('  type: COMPONENT_TYPE_VTX\n')
                out.write('  vtx {\n')
                vtx = comp['vtx']
                if 'ecosystemId' in vtx: out.write(f'    ecosystem_id: "{vtx["ecosystemId"]}"\n')
                if 'inputVoltageMinV' in vtx: out.write(f'    input_voltage_min_v: {vtx["inputVoltageMinV"]}\n')
                if 'inputVoltageMaxV' in vtx: out.write(f'    input_voltage_max_v: {vtx["inputVoltageMaxV"]}\n')
                if 'maxPowerMw' in vtx: out.write(f'    max_power_mw: {vtx["maxPowerMw"]}\n')
                if 'antennaCount' in vtx: out.write(f'    antenna_count: {vtx["antennaCount"]}\n')
                out.write('  }\n')
            elif 'motor' in comp:
                out.write('  type: COMPONENT_TYPE_MOTOR\n')
                out.write('  motor {\n')
                motor = comp['motor']
                if 'kvRating' in motor: out.write(f'    kv_rating: {motor["kvRating"]}\n')
                if 'statorDiameterMm' in motor: out.write(f'    stator_diameter_mm: {motor["statorDiameterMm"]}\n')
                if 'statorHeightMm' in motor: out.write(f'    stator_height_mm: {motor["statorHeightMm"]}\n')
                out.write('  }\n')
            elif 'frame' in comp:
                out.write('  type: COMPONENT_TYPE_FRAME\n')
                out.write('  frame {\n')
                frame = comp['frame']
                if 'wheelbaseMm' in frame: out.write(f'    wheelbase_mm: {frame["wheelbaseMm"]}\n')
                if 'maxPropSizeMm' in frame: out.write(f'    max_prop_size_mm: {frame["maxPropSizeMm"]}\n')
                out.write('  }\n')
            elif 'propeller' in comp:
                out.write('  type: COMPONENT_TYPE_PROPELLER\n')
                out.write('  propeller {\n')
                propeller = comp['propeller']
                if 'diameterMm' in propeller: out.write(f'    diameter_mm: {propeller["diameterMm"]}\n')
                if 'pitchMm' in propeller: out.write(f'    pitch_mm: {propeller["pitchMm"]}\n')
                if 'bladeCount' in propeller: out.write(f'    blade_count: {propeller["bladeCount"]}\n')
                out.write('  }\n')
            elif 'flightController' in comp:
                out.write('  type: COMPONENT_TYPE_FLIGHT_CONTROLLER\n')
                out.write('  flight_controller {\n')
                fc = comp['flightController']
                if 'mcuProcessor' in fc: out.write(f'    mcu_processor: "{fc["mcuProcessor"]}"\n')
                out.write('  }\n')
            elif 'esc' in comp:
                out.write('  type: COMPONENT_TYPE_ESC\n')
                out.write('  esc {\n')
                esc = comp['esc']
                if 'continuousCurrentA' in esc: out.write(f'    continuous_current_a: {esc["continuousCurrentA"]}\n')
                if 'burstCurrentA' in esc: out.write(f'    burst_current_a: {esc["burstCurrentA"]}\n')
                out.write('  }\n')
            elif 'camera' in comp:
                out.write('  type: COMPONENT_TYPE_CAMERA\n')
                out.write('  camera {\n')
                cam = comp['camera']
                if 'ecosystemId' in cam: out.write(f'    ecosystem_id: "{cam["ecosystemId"]}"\n')
                out.write('  }\n')
            elif 'receiver' in comp:
                out.write('  type: COMPONENT_TYPE_RECEIVER\n')
                out.write('  receiver {\n')
                rx = comp['receiver']
                if 'outputProtocol' in rx: out.write(f'    output_protocol: "{rx["outputProtocol"]}"\n')
                out.write('  }\n')
            elif 'gps' in comp:
                out.write('  type: COMPONENT_TYPE_GPS\n')
                out.write('  gps {\n')
                gps = comp['gps']
                if 'chipset' in gps: out.write(f'    chipset: "{gps["chipset"]}"\n')
                out.write('  }\n')
            elif 'antenna' in comp:
                out.write('  type: COMPONENT_TYPE_ANTENNA\n')
                out.write('  antenna {\n')
                ant = comp['antenna']
                if 'antennaStyle' in ant: out.write(f'    antenna_style: "{ant["antennaStyle"]}"\n')
                if 'connectorId' in ant: out.write(f'    connector_id: "{ant["connectorId"]}"\n')
                out.write('  }\n')

            out.write('}\n\n')

if __name__ == '__main__':
    convert()
