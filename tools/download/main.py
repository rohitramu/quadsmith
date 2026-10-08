# /// script
# requires-python = ">=3.10"
# dependencies = [
#     "beautifulsoup4",
#     "curl-cffi",
# ]
# ///
import argparse
import json
import os
import re
import sys
import time
import urllib.parse
from concurrent.futures import ThreadPoolExecutor, as_completed

from bs4 import BeautifulSoup
from curl_cffi import requests

DEFAULT_PARALLELISM = 10

HEADERS = {
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
    "Accept-Language": "en-US,en;q=0.9",
    "Accept-Encoding": "gzip, deflate, br",
    "sec-ch-ua": '"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"',
    "sec-ch-ua-mobile": "?0",
    "sec-ch-ua-platform": '"Windows"',
}

SHOPIFY_DOMAINS = [
    "pyrodrone.com",
    "www.racedayquads.com",
    "betafpv.com",
    "flywoo.net",
    "www.defiancerc.com",
    "sub250.com",
    "geprc.com",
    "rotorriot.com",
    "darwinfpv.com",
    "newbeedrone.com",
    "caddxfpv.com",
    "emax-usa.com",
]


def search_retailers(query, session):
    for domain in SHOPIFY_DOMAINS:
        url = f"https://{domain}/search/suggest.json?q={urllib.parse.quote(query)}&resources[type]=product"
        try:
            response = session.get(
                url, headers=HEADERS, impersonate="chrome110", timeout=8
            )
            if response.status_code == 200:
                data = response.json()
                products = (
                    data.get("resources", {}).get("results", {}).get("products", [])
                )

                query_parts = query.lower().split()
                best_match = None
                best_score = 0

                for product in products:
                    title = product.get("title", "").lower()

                    score = sum(1 for part in query_parts if part in title)
                    if score > best_score:
                        best_score = score
                        best_match = product

                if best_match and best_score >= min(2, len(query_parts) * 0.5):
                    return f"https://{domain}{best_match['url']}"

        except Exception:
            pass

    return ""


def fetch_page_text(url, session):
    try:
        resp = session.get(url, headers=HEADERS, impersonate="chrome110", timeout=15)
        if resp.status_code != 200:
            return ""

        soup = BeautifulSoup(resp.text, "html.parser")
        for tag in soup(["script", "style", "noscript", "svg"]):
            tag.decompose()

        text = " ".join(soup.stripped_strings)
        return text
    except Exception as e:
        return ""


def clean_query(mfg, name):
    name_lower = name.lower()
    mfg_lower = mfg.lower()

    if name_lower.startswith(mfg_lower):
        name = name[len(mfg) :].strip()

    query = f"{mfg} {name}"

    # Strip exactly " - Official Store" first
    query = re.compile(r"\s*-\s*official\s*store", re.IGNORECASE).sub("", query)
    name = re.compile(r"\s*-\s*official\s*store", re.IGNORECASE).sub("", name)

    fluff = [
        "brushless",
        "motor",
        "motors",
        "fpv",
        "drone",
        "quadcopter",
        "series",
        "racing",
        "ultralight",
        "micro",
        "v2",
        "v3",
        "se",
        "blushless",
        "seawater-proof",
    ]
    for word in fluff:
        query = re.compile(r"\b" + re.escape(word) + r"\b", re.IGNORECASE).sub(
            "", query
        )
        name = re.compile(r"\b" + re.escape(word) + r"\b", re.IGNORECASE).sub("", name)

    query = " ".join(query.split())
    name = " ".join(name.split())
    return query, name


def process_item(item, session):
    out_dir = os.path.join("..", "..", ".tmp", "raw_specs", item["type"])
    os.makedirs(out_dir, exist_ok=True)

    out_file = os.path.join(out_dir, f"{item['id']}.txt")
    meta_file = os.path.join(out_dir, f"{item['id']}.json")

    if os.path.exists(out_file):
        return f"Skipped (already exists): {item['manufacturer']} {item['name']}"

    query_mfg_name, query_name_only = clean_query(item["manufacturer"], item["name"])

    found_url = search_retailers(query_mfg_name, session)
    if not found_url and query_name_only:
        found_url = search_retailers(query_name_only, session)

    if not found_url:
        return f"Search error (no results) for {item['manufacturer']} {item['name']}"

    page_text = fetch_page_text(found_url, session)

    if len(page_text) > 200:
        with open(out_file, "w", encoding="utf-8") as f:
            f.write(page_text)

        meta_data = {
            "id": item["id"],
            "manufacturer": item["manufacturer"],
            "name": item["name"],
            "url": found_url,
        }
        with open(meta_file, "w", encoding="utf-8") as f:
            json.dump(meta_data, f, indent=2)

        return f"Saved {item['manufacturer']} {item['name']} ({len(page_text)} chars)"

    return f"Failed to fetch page for {item['manufacturer']} {item['name']} from {found_url}"


def collect_items():
    files = {
        "motors": "../../src/backend/db/seeds/motors.textproto",
        "batteries": "../../src/backend/db/seeds/batteries.textproto",
        "frames": "../../src/backend/db/seeds/frames.textproto",
        "escs": "../../src/backend/db/seeds/escs.textproto",
        "flight_controllers": "../../src/backend/db/seeds/flight_controllers.textproto",
        "propellers": "../../src/backend/db/seeds/propellers.textproto",
        "receivers": "../../src/backend/db/seeds/receivers.textproto",
        "cameras": "../../src/backend/db/seeds/cameras.textproto",
        "video_transmitters": "../../src/backend/db/seeds/video_transmitters.textproto",
        "antennas": "../../src/backend/db/seeds/antennas.textproto",
    }

    items = []
    block_regex = re.compile(r"(?s)\w+:\s*\{.*?\}")
    id_regex = re.compile(r'id:\s*"([^"]+)"')
    mfg_regex = re.compile(r'manufacturer:\s*"([^"]+)"')
    name_regex = re.compile(r'name:\s*"([^"]+)"')

    for ctype, path in files.items():
        if not os.path.exists(path):
            continue

        with open(path, "r", encoding="utf-8") as f:
            content = f.read()

        blocks = block_regex.findall(content)
        for block in blocks:
            id_match = id_regex.search(block)
            mfg_match = mfg_regex.search(block)
            name_match = name_regex.search(block)

            if id_match and mfg_match and name_match:
                items.append(
                    {
                        "type": ctype,
                        "id": id_match.group(1),
                        "manufacturer": mfg_match.group(1),
                        "name": name_match.group(1),
                    }
                )

    return items


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--parallelism", type=int, default=DEFAULT_PARALLELISM)
    args = parser.parse_args()

    items = collect_items()
    print(f"Found {len(items)} items to process.")

    session = requests.Session(impersonate="chrome110")

    if args.parallelism > 1:
        print(f"Running with {args.parallelism} threads across internal Retailer APIs.")
        completed = 0
        with ThreadPoolExecutor(max_workers=args.parallelism) as executor:
            futures = {
                executor.submit(process_item, item, session): item for item in items
            }
            for future in as_completed(futures):
                completed += 1
                try:
                    res = future.result()
                    print(f"[{completed}/{len(items)}] {res}")
                except Exception as e:
                    print(f"[{completed}/{len(items)}] Error: {e}")
    else:
        print("Running single-threaded.")
        for i, item in enumerate(items, 1):
            res = process_item(item, session)
            print(f"[{i}/{len(items)}] {res}")

    print("Done downloading raw specs!")


if __name__ == "__main__":
    main()
