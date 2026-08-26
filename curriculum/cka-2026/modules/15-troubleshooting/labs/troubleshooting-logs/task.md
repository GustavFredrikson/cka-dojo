# Diagnose a failing container from its output

Deployment `collector` in namespace `dojo-troubleshooting-logs` does not become
Ready. Preserve both containers, identify the failure from container state and
output, then make container `reporter` run continuously and print `healthy` at
least once every 10 seconds.
