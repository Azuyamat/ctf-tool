FROM ctfd/ctfd:3.8.8

COPY --chown=1001:1001 seed.py /opt/seed/seed.py
COPY --chown=1001:1001 --chmod=755 entrypoint.sh /opt/seed/entrypoint.sh
COPY --chown=1001:1001 plugins/no_tokens /opt/CTFd/CTFd/plugins/no_tokens

ENTRYPOINT ["/opt/seed/entrypoint.sh"]
