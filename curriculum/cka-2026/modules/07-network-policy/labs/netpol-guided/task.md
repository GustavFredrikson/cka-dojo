# Correct a known source-selector mismatch

In namespace `dojo-netpol-guided`, server and Service are healthy. Default deny
is intentional. NetworkPolicy `allow-client` selects the correct destination
and port, but its source Pod selector is wrong.

Correct it so `role=client` can reach `app=server` on TCP 80.
