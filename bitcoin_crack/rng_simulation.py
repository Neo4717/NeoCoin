import hashlib
import time
from weak_rng_audit import get_address_from_privkey
import json
import os
import multiprocessing

def check_timestamp_range(args):
    start_ts, end_ts, step, target_addresses = args
    found = []
    for ts in range(start_ts, end_ts, step):
        seed = str(ts).encode()
        priv_hex = hashlib.sha256(seed).hexdigest()
        try:
            addr = get_address_from_privkey(priv_hex)
            if addr in target_addresses:
                found.append((ts, addr))
        except:
            continue
    return found

def simulate_parallel(target_addresses, year=2009, step=60):
    print(f"Parallel simulation for {year} (step={step})...")
    start_ts = int(time.mktime(time.strptime(f"{year}-01-01", "%Y-%m-%d")))
    end_ts = int(time.mktime(time.strptime(f"{year+1}-01-01", "%Y-%m-%d")))
    
    num_cpus = multiprocessing.cpu_count()
    chunk_size = (end_ts - start_ts) // num_cpus
    
    ranges = []
    for i in range(num_cpus):
        s = start_ts + (i * chunk_size)
        e = s + chunk_size if i < num_cpus - 1 else end_ts
        ranges.append((s, e, step, target_addresses))
        
    with multiprocessing.Pool(processes=num_cpus) as pool:
        results = pool.map(check_timestamp_range, ranges)
        
    all_found = [item for sublist in results for item in sublist]
    for ts, addr in all_found:
        print(f"!!! ALERT: Match found! Seed: {ts} -> Address: {addr}")
    
    if not all_found:
        print(f"No matches for {year}.")

if __name__ == "__main__":
    list_path = 'bitcoin_crack/dormant_list.json'
    if os.path.exists(list_path):
        with open(list_path, 'r') as f:
            targets = set(json.load(f)) # Set for O(1) lookup
            simulate_parallel(targets, 2009, step=3600) # 1 hour granularity
            simulate_parallel(targets, 2010, step=3600)
    else:
        print("Target list missing.")
