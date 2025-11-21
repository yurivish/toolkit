watch cmd:
    fd --extension go --extension css --extension js --extension sql | entr -ccr {{ cmd }}
