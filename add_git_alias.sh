#!/bin/bash

zsh_config_file=~/.zshrc

echo '### git alias

alias ga="git add"
alias gaa="git add ."
alias gcm="git commit -m"
alias gp="git push"
alias gpl="git pull"
alias gk="git checkout"
alias gkb="git checkout -b"
alias gs="git status"
alias gsl="git stash list"
alias gsp="git stash push"
alias gspop="git stash pop"
alias gf="git reflog"
' >> $zsh_config_file

echo "added git alias to zsh"
