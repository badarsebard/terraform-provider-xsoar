set -x

sudo apt-get update
sudo apt-get -y install ca-certificates curl gnupg lsb-release makeself
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get -y install docker-ce docker-ce-cli containerd.io
sudo /vagrant/local/installer.sh -- -multi-tenant -y -elasticsearch-url=http://elastic.xsoar.local:9200 -tools=false
sudo cp /vagrant/local/demisto.lic /tmp/demisto.lic
sudo chown demisto:demisto /tmp/demisto.lic
sudo cp /tmp/demisto.lic /var/lib/demisto/
sudo systemctl restart demisto

# set cookies
until curl --fail --cookie-jar cookie.txt 'https://main.xsoar.local/health' --insecure
do
  sleep 5
done

# set admin password
XSRF=$(cat cookie.txt | grep XSRF | cut -f 7)
curl --cookie cookie.txt --cookie-jar cookie.txt 'https://main.xsoar.local/login' \
  -H "X-XSRF-TOKEN: ${XSRF}" \
  -H "Content-Type: application/json" \
  --data-raw '{"user":"admin","password":"admin","newPassword":"Xsoar123","passwordValidator":"Xsoar123","loginFailed":true,"passwordExpired":true,"weakNewPassword":false,"reusedOldPassword":false,"userLockedOut":false,"userDisabled":false,"selfUnlockRemainingMinutes":0,"loading":false,"duoFactor":"","passcode":"","shouldShowRegularLogin":false,"userTimeZone":"America/New_York"}' \
  --insecure

# create api key
curl -u "admin:Xsoar123" --cookie cookie.txt --cookie-jar cookie.txt 'https://main.xsoar.local/apikeys' \
  -H "X-XSRF-TOKEN: ${XSRF}" \
  -H "Content-Type: application/json" \
  --data-raw '{"name":"tf","apikey":"93C67EB97F30E6464DF1E5737F0470E0"}' \
  --insecure

# set dark mode
curl --cookie cookie.txt 'https://main.xsoar.local/user/update/preferences' \
  -H 'Authorization: 93C67EB97F30E6464DF1E5737F0470E0' \
  -H "Content-Type: application/json" \
  --data-raw '{"id":"admin","homepage":"","investigationPage":"","disableHyperSearch":false,"editorStyle":"","timeZone":"","hours24":"","dateFormat":"","theme":"dark","helpSnippetDisabled":false,"shortcutsDisabled":false,"notificationsSettings":{"email":{"all":true},"pushNotifications":{"all":true}}}' \
  --insecure