import os
import time
from .errors_counters import log_shut_down


def run_timer(main, wait_x):
    """Run the main function periodically or once depending on environment.
    
    In Kubernetes/job mode, just returns (main has already run once).
    In container mode, runs in an infinite loop with periodic restarts.
    """
    run_once = os.getenv('RUN_ONCE', 'false').lower() in ('true', '1', 'yes')
    
    if run_once:
        # Kubernetes job mode: main() has already executed, just exit
        print("Running in job mode (RUN_ONCE=true). Job complete.")
        return
    
    # Container mode: Keep running with periodic restarts
    while True:
        log_shut_down()
        wait_time = int(wait_x) * 3600
        print(f"Waiting for {wait_time} seconds before restarting...")
        time.sleep(wait_time)
        main()

def wait_for_server(wait_t):
    print("Waiting for server to initialize...")
    time.sleep(wait_t)
