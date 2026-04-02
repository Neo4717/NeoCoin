import requests
from bs4 import BeautifulSoup
import json
import time
import re

def scrape_bitinfocharts_dormant(limit=1000):
    addresses = []
    # BitInfoCharts has pages of 100 addresses
    # https://bitinfocharts.com/top-100-dormant_5y-bitcoin-addresses.html
    # https://bitinfocharts.com/top-100-dormant_5y-bitcoin-addresses-2.html
    # etc.
    
    for i in range(1, (limit // 100) + 1):
        url = f"https://bitinfocharts.com/top-100-dormant_5y-bitcoin-addresses-{i}.html"
        if i == 1:
            url = "https://bitinfocharts.com/top-100-dormant_5y-bitcoin-addresses.html"
            
        print(f"Scraping page {i}...")
        headers = {
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36'
        }
        response = requests.get(url, headers=headers)
        if response.status_code != 200:
            print(f"Error: {response.status_code}")
            break
            
        soup = BeautifulSoup(response.text, 'html.parser')
        # Addresses are usually in <a> tags with a specific pattern
        # Looking at BitInfoCharts, addresses are in table rows
        table = soup.find('table', {'id': 'tblOne'})
        if not table:
            print("Table not found")
            break
            
        rows = table.find_all('tr')[1:] # Skip header
        for row in rows:
            tds = row.find_all('td')
            if len(tds) > 1:
                # The address is in the second or third column usually
                # Let's find the link that contains the address
                a_tag = tds[1].find('a')
                if a_tag:
                    addr = a_tag.text
                    # Validate it's a BTC address (starts with 1, 3, or bc1)
                    if re.match(r'^[13][a-km-zA-HJ-NP-Z1-9]{25,34}$', addr):
                        addresses.append(addr)
                    elif re.match(r'^bc1[ac-hj-np-z02-9]{11,71}$', addr):
                        addresses.append(addr)
                        
        if len(addresses) >= limit:
            break
            
        time.sleep(2) # Be polite
        
    return addresses[:limit]

if __name__ == "__main__":
    # We might need to install beautifulsoup4
    try:
        dormant_addresses = scrape_bitinfocharts_dormant(1000)
        print(f"Scraped {len(dormant_addresses)} addresses.")
        with open('/root/neocoin/bitcoin_crack/dormant_list.json', 'w') as f:
            json.dump(dormant_addresses, f, indent=4)
        print("Saved to bitcoin_crack/dormant_list.json")
    except ImportError:
        print("bs4 not found, installing...")
        import subprocess
        subprocess.check_call(["pip", "install", "beautifulsoup4"])
        # Retry
        import sys
        import os
        os.execv(sys.executable, ['python3'] + sys.argv)
