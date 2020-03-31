set -ex
# Common
sudo install -Dm440 -gscoreboardd judge.secret /etc/judge.secret
sudo install -Dm440 -gta judge.secret /etc/judge.priv
sudo setfacl -m g:ta:r /etc/judge.secret

# Judge
sudo install -Dm2711 -gscoreboardd xjudge /usr/local/bin/xjudge
for hw in hw1 lab2 hw2
do
	sudo ln -sf /usr/local/bin/xjudge /usr/local/bin/$hw-judge
done

# Scoreboard
sudo install -dm755 -oscoreboardd -gscoreboardd /home/scoreboardd/ipc20
sudo install -Dm755 -oscoreboardd -gscoreboardd sb /home/scoreboardd/ipc20/sb
