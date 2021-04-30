"""Formats the DJI restricted zones into a KML file."""

import json
import sys
from typing import List, Tuple
from dataclasses import dataclass, field


@dataclass
class Polygon:
    points: List[Tuple[float]] = field(default_factory=list)
    name: str = ""


def format_kml(in_file, out_file) -> None:
    """Produces a KML file with the DJI restrictions."""
    json_data = json.load(in_file)
    polygons: List[Polygon] = []

    for area in json_data["areas"]:
        new_polygon = Polygon()
        new_polygon.name = area["name"]
        polygon_points = area["sub_areas"][0]["polygon_points"]
        if polygon_points is not None:
            for point in polygon_points[0]:
                new_polygon.points.append(tuple(point))
            polygons.append(new_polygon)

    out_file.write(HEADER)
    for polygon in polygons:
        formatted = "\n".join((f"{' ' * 24}{point[0]},{point[1]}" for point in polygon.points))
        out_file.write(PLACEMARK_TEMPLATE.format(name=polygon.name, coordinates=formatted))
    out_file.write(FOOTER)


HEADER = """<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2" xmlns:gx="http://www.google.com/kml/ext/2.2" xmlns:kml="http://www.opengis.net/kml/2.2" xmlns:atom="http://www.w3.org/2005/Atom">
<Document>
    <name>DJI restrictions</name>
    <StyleMap id="m_ylw-pushpin">
        <Pair>
            <key>normal</key>
            <styleUrl>#s_ylw-pushpin</styleUrl>
        </Pair>
        <Pair>
            <key>highlight</key>
            <styleUrl>#s_ylw-pushpin_hl</styleUrl>
        </Pair>
    </StyleMap>
    <Style id="s_ylw-pushpin">
        <IconStyle>
            <scale>1.1</scale>
            <Icon>
                <href>http://maps.google.com/mapfiles/kml/pushpin/ylw-pushpin.png</href>
            </Icon>
            <hotSpot x="20" y="2" xunits="pixels" yunits="pixels"/>
        </IconStyle>
        <LineStyle>
            <color>ff0000ff</color>
        </LineStyle>
        <PolyStyle>
            <color>440000ff</color>
        </PolyStyle>
    </Style>
    <Style id="s_ylw-pushpin_hl">
        <IconStyle>
            <scale>1.3</scale>
            <Icon>
                <href>http://maps.google.com/mapfiles/kml/pushpin/ylw-pushpin.png</href>
            </Icon>
            <hotSpot x="20" y="2" xunits="pixels" yunits="pixels"/>
        </IconStyle>
        <LineStyle>
            <color>ff0000ff</color>
        </LineStyle>
        <PolyStyle>
            <color>00000000</color>
        </PolyStyle>
    </Style>
"""

FOOTER = """</Document>
</kml>
"""

PLACEMARK_TEMPLATE = """    <Placemark>
        <name>{name}</name>
        <styleUrl>#m_ylw-pushpin</styleUrl>
        <Polygon>
            <tessellate>1</tessellate>
            <outerBoundaryIs>
                <LinearRing>
                    <coordinates>
{coordinates}
                    </coordinates>
                </LinearRing>
            </outerBoundaryIs>
        </Polygon>
    </Placemark>
"""

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print(f"Usage: python {sys.argv[0]} <dji-restrictions.json> <out-file-name.kml>")
        sys.exit(1)

    with open(sys.argv[1], "r") as dji:
        with open(sys.argv[2], "w") as out_kml:
            format_kml(dji, out_kml)
