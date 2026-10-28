import os

from flask import redirect, request


def load(app):
    if os.environ.get("CTFD_DISABLE_TOKENS") != "1":
        return

    @app.before_request
    def reject_tokens():
        if request.headers.get("Authorization", "").startswith("Token "):
            return redirect("/login")
