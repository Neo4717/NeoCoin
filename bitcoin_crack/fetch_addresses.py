import requests
import json
import time

def fetch_dormant_addresses(limit=1000):
    base_url = "https://api.blockchair.com/bitcoin/addresses"
    # 1 BTC = 100,000,000 satoshis
    # Dormant: last seen spending before 2013-01-01
    # Note: last_seen_spending can be null if never spent
    
    addresses = []
    page_size = 100
    pages = limit // page_size
    
    for i in range(pages):
        offset = i * page_size
        # Query for balance >= 1 BTC and last_seen_spending before 2013-01-01 (including null)
        # We use OR logic for null: last_seen_spending(..2013-01-01) might not include nulls in some APIs, 
        # but Blockchair's documentation says last_seen_spending(null) is for those that never spent.
        # Let's try to get those that never spent and were first seen before 2013.
        
        # Actually, let's just use first_seen_receiving(..2013-01-01) and last_seen_spending(..2013-01-01)
        # to ensure they are old and haven't moved recently.
        query = f"q=balance(100000000..),first_seen_receiving(..2013-01-01),last_seen_spending(..2013-01-01)&limit={page_size}&offset={offset}&s=balance(desc)"
        url = f"{base_url}?{query}"
        
        print(f"Fetching page {i+1}/{pages}...")
        response = requests.get(url)
        if response.status_code == 200:
            data = response.json()
            page_addresses = [item['address'] for item in data['data']]
            addresses.extend(page_addresses)
            if len(page_addresses) < page_size:
                break
        else:
            print(f"Error fetching data: {response.status_code}")
            break
        
        # Be nice to the API
        time.sleep(1)
        
    return addresses[:limit]

if __name__ == "__main__":
    dormant_addresses = fetch_dormant_addresses(1000)
    print(f"Fetched {len(dormant_addresses)} addresses.")
    with open('/root/neocoin/bitcoin_crack/dormant_list.json', 'w') as f:
        json.dump(dormant_addresses, f, indent=4)
    print("Saved to bitcoin_crack/dormant_list.json")
