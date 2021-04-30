"""Converts APRS readings to a KML file."""

from dataclasses import dataclass
from typing import List, Optional
import datetime
import requests
import re
import sys

try:
    raise ImportError
    import aprslib
    parse_aprs = lambda x: aprslib.parse
except ImportError:
    parse_aprs = None


def get_packets_html(callsign: str, login_cookie: str) -> str:
    """Returns the HTML of the packets for a given callsign."""
    response = requests.get(
        "https://aprs.fi/",
        params={
            "c": "raw",
            "call": callsign,
            "limit": 50,
            "view": "normal",
        },
        headers={
            "Host": "aprs.fi",
            "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:87.0) Gecko/20100101 Firefox/87.0",
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
            "Accept-Language": "en-US,en;q=0.5",
            "DNT": "1",
            "Cookie": login_cookie,
        }
    )
    response.raise_for_status()
    return response.text


HTML_TAG_RE = re.compile(r"<[^>]+>")
def get_messages_from_html(html: str) -> List[str]:
    """Returns a list of APRS messages from HTML."""
    messages: List[str] = []
    for line in html.split("\n"):
        stripped = line.strip()
        # We want both 'raw_line' and 'raw_line_error' classes
        if stripped.startswith("<span class='raw_line"):
            message = HTML_TAG_RE.sub("", stripped)
            message = message.split(" ")[3]
            message = message.replace("&gt;", ">")
            message = message.replace("\xa0", " ")
            # We don't need to worry about "[Rate limited" at the end because
            # the split on " " above will remove it
            messages.append(message)

    return messages


@dataclass
class Point:
    latitude_d: float
    longitude_d: float
    altitude_m: Optional[float]


def get_points_from_messages(messages: List[str]) -> List[Point]:
    """Returns parsed points from a list of APRS messages."""
    global parse_aprs
    if parse_aprs is None:
        parse_aprs = _parse_aprs

    points: List[Point] = []
    for message in messages:
        parsed = parse_aprs(message)
        # Cars often don't include altitude, and I have been using
        # them for testing. So if it's not there, just drop it.
        altitude_m = parsed.get("altitude")
        point = Point(
            latitude_d=parsed["latitude"],
            longitude_d=parsed["longitude"],
            altitude_m=altitude_m,
        )
        points.append(point)

    return points


def save_kml(file, points: List[Point], name: str = None) -> None:
    """Saves a KML file."""
    if name is None:
        name = "APRS path"
    beginning = f"""<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2">
<Document>
<name>{name}</name>
<description><![CDATA[APRS path]]></description>
<Style id="yellowPoly">
<LineStyle>
<color>7f00ffff</color>
<width>4</width>
</LineStyle>
<PolyStyle>
<color>7f00ff00</color>
</PolyStyle>
</Style>
<Placemark>
<name>Flight path</name>
<styleUrl>#yellowPoly</styleUrl>
<LineString>
<extrude>1</extrude>
<tesselate>1</tesselate>
<altitudeMode>absolute</altitudeMode>
<coordinates>
"""

    ending = """</coordinates>
</LineString>
</Placemark>
</Document>
</kml>
"""
    formatted: List[str] = []
    for point in points:
        string = f"{point.longitude_d},{point.latitude_d}"
        if point.altitude_m is not None:
            string += f",{point.altitude_m}"
        formatted.append(string)

    coordinates = "\n".join(formatted)
    file.write(f"{beginning}{coordinates}{ending}")


def base91_to_decimal(text: str) -> float:
    """Poor man's base 91."""
    if re.findall(r"[\x00-\x20\x7c-\xff]", text):
        raise ValueError("invalid character in sequence")

    text = text.lstrip("!")
    decimal = 0
    length = len(text) - 1
    for i, char in enumerate(text):
        decimal += (ord(char) - 33) * (91 ** (length - i))

    return decimal if text != "" else 0    


def _parse_aprs(body: str) -> dict:
    """Poor man's APRS parser."""
    def parse_minutes(minutes: str) -> float:
        """Parse minutes into degrees, e.g. 4857.89 (representing 48d57.89m)"""
        if len(minutes.split(".")[0]) == 4:
            split = 2
        else:
            split = 3
        return float(minutes[:split]) + float(minutes[split:]) / 60

    parsed = {}
    match = re.search(r"\d{6}h(\d{4}\.\d{2})([NS])/(\d{5}\.\d{2})([EW]).+A=(\d+)", body)
    if match:
        # Long format
        latitude_str, n_s, longitude_str, e_w, altitude_str = match.groups()
        latitude_d = parse_minutes(latitude_str)
        longitude_d = parse_minutes(longitude_str)
        altitude_m = float(altitude_str)
        parsed.update({"altitude": altitude_m})
        if n_s.lower() == "s":
            latitude_d *= -1
        if e_w.lower() == "s":
            longitude_d *= -1

    elif re.match(r"^[\/\\A-Za-j][!-|]{8}[!-{}][ -|]{3}", body):
        if len(body) < 13:
            raise ParseError("Invalid compressed packet (less than 13 characters)")

        parsed.update({"format": "compressed"})

        compressed = body[:13]
        body = body[13:]

        symbol_table = compressed[0]
        symbol = compressed[9]

        try:
            latitude_d = 90 - (base91_to_decimal(compressed[1:5]) / 380926.0)
            longitude_d = -180 + (base91_to_decimal(compressed[5:9]) / 190463.0)
        except ValueError:
            raise ParseError("invalid characters in latitude/longitude encoding")

        # parse csT

        # converts the relevant characters from base91
        c1, s1, ctype = [ord(x) - 33 for x in compressed[10:13]]

        if c1 == -1:
            parsed.update({"gpsfixstatus": 1 if ctype & 0x20 == 0x20 else 0})

        if -1 in [c1, s1]:
            pass
        elif ctype & 0x18 == 0x10:
            parsed.update({"altitude": (1.002 ** (c1 * 91 + s1)) * 0.3048})
        elif c1 >= 0 and c1 <= 89:
            parsed.update({"course": 360 if c1 == 0 else c1 * 4})
            parsed.update({"speed": (1.08 ** s1 - 1) * 1.852})  # mul = convert knts to kmh
        elif c1 == 90:
            parsed.update({"radiorange": (2 * 1.08 ** s1) * 1.609344})  # mul = convert mph to kmh

        parsed.update({
            "symbol": symbol,
            "symbol_table": symbol_table,
        })

    parsed.update({
        "latitude": latitude_d,
        "longitude": longitude_d,
    })

    return parsed


def main() -> None:
    """Main."""
    if len(sys.argv) != 2:
        print("Usage:")
        print(f"{sys.argv[0]} <callsign, e.g. KE0FZV-1>")
        print(f"{sys.argv[0]} <aprs.fi HTML file>")

    callsign: Optional[str] = None
    if sys.argv[1].endswith(".html"):
        print("Processing downloaded HTML")
        with open(sys.argv[1], "r") as file:
            html = file.read()
    else:
        callsign = sys.argv[1]
        print(f"Downloading HTML from aprs.fi for {callsign}")
        with open("cookie.txt", "r") as file:
            # The login cookie should look like this:
            # mapssession=<stuff>; mapsset=<stuff>; size=1080x886
            # You can get this from Firefox's network inspector
            cookie = file.read().strip()

        html = get_packets_html(callsign, cookie)
        # Let's save the file too, for testing
        with open("raw.html", "w") as file:
            file.write(html)

    messages = get_messages_from_html(html)
    points = get_points_from_messages(messages)
    if len(points) == 0:
        print("No points found, aborting")
        return

    now = datetime.datetime.now()
    date_stamp = datetime.datetime.strftime(now, "%Y-%m-%d-%H-%M")
    if callsign is None:
        file_name = f"{date_stamp}.kml"
    else:
        file_name = f"{callsign}-{date_stamp}.kml"

    print(f"Saving to {file_name}")
    with open(file_name, "w") as file:
        save_kml(file, points, callsign)


if __name__ == "__main__":
    main()
