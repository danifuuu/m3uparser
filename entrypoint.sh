#!/bin/bash

# Set defaults for PUID and PGID if not specified
PUID=${PUID:-1000}
PGID=${PGID:-1000}

# Remove the user and group if they exist
echo "Attempting to delete existing user and group..."
userdel m3uuser >/dev/null 2>&1
groupdel m3ugroup >/dev/null 2>&1

# Create the group
echo "Creating new group 'm3ugroup'..."
groupadd -g "$PGID" m3ugroup

# Create the user
echo "Creating new user 'm3uuser'..."
useradd -u "$PUID" -g "$PGID" --no-log-init m3uuser

# Set ownership and permissions
echo "Setting ownership and permissions..."
chown -R "$PUID:$PGID" /usr/src/app
chmod +x /usr/src/app/parser/parser_script.py

# Check if running as Kubernetes job
if [ "${KUBERNETES_JOB:-false}" = "true" ] || [ "${RUN_ONCE:-false}" = "true" ]; then
    echo "Running as Kubernetes job (RUN_ONCE mode)..."
    RUN_ONCE_MODE=true
else
    echo "Running as continuous container..."
    RUN_ONCE_MODE=false
fi

# Switch to the m3uuser and run parser_script.py
echo "Switching to user 'm3uuser' and running parser_script..."
su -s /bin/bash -c "RUN_ONCE=$RUN_ONCE_MODE exec python3 /usr/src/app/parser/parser_script.py" m3uuser

# Capture exit code
EXIT_CODE=$?

# Exit the entrypoint script
echo "Entrypoint script execution completed."
exit $EXIT_CODE

