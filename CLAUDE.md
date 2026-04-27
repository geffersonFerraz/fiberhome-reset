Precisamos fazer um script para resetar um dispositivo fiber home.

1. O router fica na porta 192.168.1.1:8090
1.1 Faça o teste para ver se a pagina de login é carregada.
1.2 Se não carregar, informe o erro.
2. Se a pagina carregar, faça o login utilizando o usuario e senha a seguir: admin|%0|F?H@f!berhO3e
2.1 Esse é um curl de exemplo do login, é importante que salvemos o cookie retornado nos headers response:
curl 'http://192.168.1.1:8090/goform/webLogin' \
  -X POST \
  -H 'User-Agent: Mozilla/5.0 (X11; Linux x86_64; rv:150.0) Gecko/20100101 Firefox/150.0' \
  -H 'Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8' \
  -H 'Accept-Language: pt-BR' \
  -H 'Accept-Encoding: gzip, deflate' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -H 'Origin: http://192.168.1.1' \
  -H 'Connection: keep-alive' \
  -H 'Referer: http://192.168.1.1/login_inter.asp' \
  -H 'Cookie: fhstamp=LaPd6mticnd7V9pp0jV92wSbS3Q1rv1tt6Dyw' \
  -H 'Upgrade-Insecure-Requests: 1' \
  -H 'Priority: u=4' \
  -H 'Pragma: no-cache' \
  -H 'Cache-Control: no-cache' \
  --data-raw 'username=YWRtaW4%3D&password=JTB8Rj9IQGYhYmVyaE8zZQ%3D%3D'

  Header responde com cookie:
  Set-Cookie
	fhstamp=faRdgmeiGn9jyEbnFNNaT5K51L9lhMgWrJ274;path=/;HttpOnly;

2.2 Se não conseguir logar, informe o erro.
3. Agora vamos chamar o curl para resetar:
curl 'http://192.168.1.1:8090/goform/adminRestore' \
  -X POST \
  -H 'User-Agent: Mozilla/5.0 (X11; Linux x86_64; rv:150.0) Gecko/20100101 Firefox/150.0' \
  -H 'Accept: */*' \
  -H 'Accept-Language: pt-BR' \
  -H 'Accept-Encoding: gzip, deflate' \
  -H 'Content-type: application/x-www-form-urlencoded' \
  -H 'Origin: http://192.168.1.1:8090' \
  -H 'Connection: keep-alive' \
  -H 'Referer: http://192.168.1.1:8090/management/adminRestore.asp' \
  -H 'Cookie: fhstamp=LaPd6mticnd7V9pp0jV92wSbS3Q1rv1tt6Dyw' \
  -H 'Priority: u=0' \
  -H 'Pragma: no-cache' \
  -H 'Cache-Control: no-cache' \
  --data-raw 'n/a&x-csrftoken=LaPd6mticnd7V9pp0jV92wSbS3Q1rv1tt6Dyw'
