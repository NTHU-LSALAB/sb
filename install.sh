set -ex
# Common
sudo install -Dm440 -gta /dev/null /etc/judge.priv
sudo install -dm750 -oscoreboardd -gscoreboardd /run/scoreboard

# Judge
sudo install -Dm2711 -gscoreboardd xjudge /usr/local/bin/xjudge
for hw in hw1 lab2 hw2
do
	sudo ln -sf /usr/local/bin/xjudge /usr/local/bin/$hw-judge
done

# Scoreboard
sudo install -dm755 -oscoreboardd -gscoreboardd /home/scoreboardd/ipc20
sudo install -Dm755 -oscoreboardd -gscoreboardd sb /home/scoreboardd/ipc20/sb
