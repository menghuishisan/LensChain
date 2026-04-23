-- 回滚模块05 CTF竞赛参数化漏洞模板库基础种子。

DELETE FROM challenge_templates
WHERE code IN (
    'reentrancy-template',
    'flash-loan-template',
    'integer-overflow-template',
    'access-control-template',
    'oracle-manipulation-template',
    'unchecked-call-template',
    'delegatecall-injection-template',
    'selfdestruct-template',
    'tx-origin-template',
    'amm-price-manipulation-template',
    'uninitialized-proxy-template',
    'signature-replay-template',
    'front-running-template',
    'storage-collision-template',
    'randomness-template',
    'denial-of-service-template',
    'approval-race-template',
    'erc777-hook-template',
    'bridge-validation-template',
    'governance-attack-template'
);
