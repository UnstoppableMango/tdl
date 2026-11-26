# shellcheck shell=bash
# gomod2nix writes no vendor/modules.txt, and without one the build info
# records no dependency. Each package directory under vendor/ belongs to the
# longest module path go.mod requires that holds it.
writeVendorModules() {
  chmod u+w vendor
  go mod edit -json | jq -r '.Require[] | "\(.Path) \(.Version)"' | sort >"$TMPDIR/requires"
  (cd vendor && find -L . -name '*.go' -printf '%h\n' | sed 's#^\./##' | sort -u) >"$TMPDIR/packages"
  awk '
    NR == FNR { mods[++n] = $1; version[$1] = $2; next }
    {
      best = ""
      for (m in version)
        if (($0 == m || index($0, m "/") == 1) && length(m) > length(best)) best = m
      if (best != "") pkgs[best] = pkgs[best] $0 "\n"
    }
    END {
      for (i = 1; i <= n; i++) {
        # The go line of each module go.mod sets its language version.
        lang = ""
        file = "vendor/" mods[i] "/go.mod"
        while ((getline line < file) > 0) if (line ~ /^go [0-9]/) { split(line, f, " "); lang = "; go " f[2] }
        close(file)
        printf "# %s %s\n## explicit%s\n%s", mods[i], version[mods[i]], lang, pkgs[mods[i]]
      }
    }
  ' "$TMPDIR/requires" "$TMPDIR/packages" >vendor/modules.txt
}
