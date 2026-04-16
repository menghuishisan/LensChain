<?php
$team = getenv('CTF_TEAM_ID') ?: 'unknown';
$flag = getenv('CTF_FLAG') ?: 'flag{demo-placeholder}';
?><!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <title>LensChain CTF Web Target</title>
</head>
<body>
  <h1>LensChain CTF Web Target</h1>
  <p>当前队伍：<?php echo htmlspecialchars($team, ENT_QUOTES, 'UTF-8'); ?></p>
  <p>本镜像为通用 Web 靶机接入模板，正式题目请替换业务代码。</p>
  <pre><?php echo htmlspecialchars($flag, ENT_QUOTES, 'UTF-8'); ?></pre>
</body>
</html>
