import datetime
import io
import os
import zipfile

from werkzeug.datastructures import FileStorage

from CTFd import create_app
from CTFd.cache import clear_challenges, clear_config, clear_standings
from CTFd.models import (
    Admins,
    Challenges,
    Flags,
    Hints,
    Solves,
    Tags,
    Teams,
    UserTokens,
    Users,
    db,
)
from CTFd.utils import get_config, set_config
from CTFd.utils.uploads import upload_file

PLAYER_TOKEN = os.getenv("CTFD_PLAYER_TOKEN", "ctfd_dev_player_token")
ADMIN_TOKEN = os.getenv("CTFD_ADMIN_TOKEN", "ctfd_dev_admin_token")
USER_MODE = os.getenv("CTFD_USER_MODE") or "teams"
NEVER = datetime.datetime(2099, 1, 1)


def zip_bytes(files):
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w") as zf:
        for name, data in files.items():
            zf.writestr(name, data)
    return buf.getvalue()


CHALLENGES = [
    {
        "key": "cookies",
        "name": "Cookies",
        "category": "web",
        "value": 100,
        "description": "Who doesn't love **cookies**? Try to figure out the best one.",
        "connection_info": "https://cookies.ctf.local",
        "flag": "flag{c00k1e_m0nst3r}",
        "tags": ["easy"],
        "hints": [{"content": "Look at the cookie named `name`.", "cost": 0}],
    },
    {
        "key": "baby-rop",
        "name": "Baby ROP",
        "category": "pwn",
        "value": 250,
        "description": "Classic stack overflow. Libc provided.",
        "connection_info": "nc localhost 31337",
        "flag": "flag{r3t_2_w1n}",
        "tags": ["medium"],
        "hints": [
            {"title": "Approach", "content": "Leak a libc address, then ret2libc.", "cost": 25},
            {"title": "Solution", "content": "Overwrite the return address at offset 40.", "cost": 1000},
        ],
        "files": {
            "baby-rop.zip": zip_bytes(
                {
                    "baby-rop/chall": b"\x7fELF placeholder binary\n",
                    "baby-rop/libc.so.6": b"\x7fELF placeholder libc\n",
                }
            )
        },
    },
    {
        "key": "rsa-1",
        "name": "RSA 1",
        "category": "crypto",
        "value": 150,
        "description": "Small e, big problems. You only get **3** attempts.",
        "flag": "flag{cub3_r00t_4tt4ck}",
        "max_attempts": 3,
        "files": {"output.txt": b"n = 1234567\ne = 3\nc = 42\n"},
    },
    {
        "key": "rsa-1-dup",
        "name": "RSA-1",
        "category": "crypto",
        "value": 200,
        "description": "Same slug as `RSA 1`, different challenge.",
        "flag": "flag{sl0g_c0ll1s10n}",
    },
    {
        "key": "sanity",
        "name": "Sanity Check",
        "category": "misc",
        "value": 50,
        "description": "The flag is `flag{welcome}`.",
        "flag": "flag{welcome}",
        "solved_by_teammate": True,
    },
    {
        "key": "symbols",
        "name": "Spaces & Symbols!? (ünïcode)",
        "category": "Misc Stuff",
        "value": 75,
        "description": "A name and category that need slugging.",
        "flag": "flag{sl0g_m3}",
    },
    {
        "key": "locked",
        "name": "Locked Away",
        "category": "forensics",
        "value": 200,
        "description": "Unlocked by solving Cookies.",
        "flag": "flag{unl0ck3d}",
        "requires": ["cookies"],
    },
    {
        "key": "hidden",
        "name": "Hidden Gem",
        "category": "rev",
        "value": 300,
        "description": "Not visible to players.",
        "flag": "flag{n0t_y3t}",
        "state": "hidden",
    },
]


def configure():
    set_config("ctf_name", "ctf-tool dev")
    set_config("ctf_description", "Local CTFd instance for developing the ctf tool")
    set_config("user_mode", USER_MODE)
    set_config("ctf_theme", "core")
    set_config("challenge_visibility", "private")
    set_config("registration_visibility", "public")
    set_config("score_visibility", "public")
    set_config("account_visibility", "public")
    set_config("verify_emails", False)
    set_config("team_size", 0)
    set_config("start", None)
    set_config("end", None)
    set_config("freeze", None)
    set_config("setup", True)


def add_user(model, name, password, team=None, verified=True):
    user = model(
        name=name,
        email=f"{name}@ctf.local",
        password=password,
        verified=verified,
    )
    if team is not None:
        user.team_id = team.id
    db.session.add(user)
    db.session.commit()
    return user


def add_token(user, value, description):
    db.session.add(
        UserTokens(
            user_id=user.id,
            value=value,
            description=description,
            expiration=NEVER,
        )
    )
    db.session.commit()


def add_challenge(spec, ids):
    chal = Challenges(
        name=spec["name"],
        category=spec["category"],
        value=spec["value"],
        description=spec["description"],
        connection_info=spec.get("connection_info"),
        max_attempts=spec.get("max_attempts", 0),
        state=spec.get("state", "visible"),
        type="standard",
    )
    if "requires" in spec:
        chal.requirements = {
            "prerequisites": [ids[key] for key in spec["requires"]],
            "anonymize": True,
        }
    db.session.add(chal)
    db.session.commit()

    db.session.add(Flags(challenge_id=chal.id, content=spec["flag"], type="static"))
    for tag in spec.get("tags", []):
        db.session.add(Tags(challenge_id=chal.id, value=tag))
    for hint in spec.get("hints", []):
        db.session.add(
            Hints(
                challenge_id=chal.id,
                title=hint.get("title"),
                content=hint["content"],
                cost=hint["cost"],
            )
        )
    db.session.commit()

    for filename, data in spec.get("files", {}).items():
        upload_file(
            file=FileStorage(stream=io.BytesIO(data), filename=filename),
            challenge_id=chal.id,
            type="challenge",
        )
    return chal


def seed():
    configure()

    admin = add_user(Admins, "admin", "admin")
    add_token(admin, ADMIN_TOKEN, "dev admin token")

    if USER_MODE == "users":
        team = rival_team = None
        player = add_user(Users, "player", "player")
        teammate = add_user(Users, "teammate", "teammate")
        rival = add_user(Users, "rival", "rival")
    else:
        team = Teams(name="hashbrowns", password="hashbrowns")
        db.session.add(team)
        db.session.commit()

        player = add_user(Users, "player", "player", team)
        teammate = add_user(Users, "teammate", "teammate", team)
        team.captain_id = player.id
        db.session.commit()

        rival_team = Teams(name="rivals", password="rivals")
        db.session.add(rival_team)
        db.session.commit()
        rival = add_user(Users, "rival", "rival", rival_team)
        rival_team.captain_id = rival.id
        db.session.commit()
    add_token(player, PLAYER_TOKEN, "dev player token")

    solver = player if team is None else teammate

    ids = {}
    for spec in CHALLENGES:
        chal = add_challenge(spec, ids)
        ids[spec["key"]] = chal.id
        if spec.get("solved_by_teammate"):
            db.session.add(
                Solves(
                    user_id=solver.id,
                    team_id=team.id if team else None,
                    challenge_id=chal.id,
                    ip="127.0.0.1",
                    provided=spec["flag"],
                )
            )
            db.session.add(
                Solves(
                    user_id=rival.id,
                    team_id=rival_team.id if rival_team else None,
                    challenge_id=chal.id,
                    ip="127.0.0.1",
                    provided=spec["flag"],
                )
            )
    db.session.commit()


def main():
    app = create_app()
    with app.app_context():
        if get_config("setup"):
            print("[seed] already seeded, skipping", flush=True)
        else:
            seed()
            clear_config()
            clear_standings()
            clear_challenges()
            print("[seed] seeded ctf-tool dev data", flush=True)

        print(f"[seed] player token: {PLAYER_TOKEN}", flush=True)
        print(f"[seed] admin token:  {ADMIN_TOKEN}", flush=True)


if __name__ == "__main__":
    main()
