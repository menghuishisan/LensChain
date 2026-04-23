-- 模块05 CTF竞赛：参数化漏洞模板库基础种子。
-- 文档依据：
-- 1. docs/modules/05-CTF竞赛/01-功能需求说明.md 5.3 参数化模板库（约20个模板）
-- 2. docs/modules/05-CTF竞赛/03-API接口设计.md 2.13-2.15 漏洞转化接口

INSERT INTO challenge_templates (
    id,
    name,
    code,
    description,
    vulnerability_type,
    base_source_code,
    base_assertions,
    base_setup_transactions,
    parameters,
    variants,
    reference_events,
    difficulty_range
) VALUES
(
    1780000000520001,
    '重入攻击模板',
    'reentrancy-template',
    '基于提款顺序错误的重入漏洞，支持单函数、跨函数、跨合约变体。',
    'reentrancy',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    uint256 public pool = {{initial_pool}} ether;
    function withdraw() external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"攻击成功后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"single-function","options":["single-function","cross-function","cross-contract"]},{"key":"initial_pool","label":"初始资金池 ETH","type":"number","default":"10"}]}'::jsonb,
    '[{"name":"单函数重入","params":{"variant":"single-function","initial_pool":"10"},"suggested_difficulty":1},{"name":"跨函数重入","params":{"variant":"cross-function","initial_pool":"30"},"suggested_difficulty":2},{"name":"跨合约重入","params":{"variant":"cross-contract","initial_pool":"50"},"suggested_difficulty":3}]'::jsonb,
    '[{"name":"The DAO","date":"2016-06","loss":"360万ETH"}]'::jsonb,
    '{"min":1,"max":4}'::jsonb
),
(
    1780000000520002,
    '闪电贷套利模板',
    'flash-loan-template',
    '基于闪电贷资金瞬时放大的套利/操纵漏洞模板。',
    'flash-loan',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    uint256 public loanAmount = {{loan_amount}} ether;
    function executeFlashLoan() external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"闪电贷利用完成后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"DEX变体","type":"select","default":"single-dex","options":["single-dex","multi-dex","oracle-linked"]},{"key":"loan_amount","label":"闪电贷额度 ETH","type":"number","default":"1000"}]}'::jsonb,
    '[{"name":"单DEX套利","params":{"variant":"single-dex","loan_amount":"1000"},"suggested_difficulty":2},{"name":"多DEX路径","params":{"variant":"multi-dex","loan_amount":"5000"},"suggested_difficulty":3},{"name":"预言机关联","params":{"variant":"oracle-linked","loan_amount":"10000"},"suggested_difficulty":4}]'::jsonb,
    '[{"name":"bZx flash loan incident","date":"2020-02","loss":"约95万美元"}]'::jsonb,
    '{"min":2,"max":5}'::jsonb
),
(
    1780000000520003,
    '整数溢出模板',
    'integer-overflow-template',
    '覆盖加法、乘法、类型转换等整数边界问题。',
    'integer-overflow',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    uint8 public counter = {{start_value}};
    function overflow(uint8 amount) external { unchecked { counter += amount; } solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"溢出路径触发后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"addition","options":["addition","multiplication","cast"]},{"key":"start_value","label":"起始值","type":"number","default":"250"}]}'::jsonb,
    '[{"name":"加法溢出","params":{"variant":"addition","start_value":"250"},"suggested_difficulty":1},{"name":"乘法溢出","params":{"variant":"multiplication","start_value":"128"},"suggested_difficulty":2},{"name":"类型转换截断","params":{"variant":"cast","start_value":"255"},"suggested_difficulty":3}]'::jsonb,
    '[{"name":"BatchOverflow","date":"2018-04","loss":"多交易所临时冻结"}]'::jsonb,
    '{"min":1,"max":3}'::jsonb
),
(
    1780000000520004,
    '权限控制模板',
    'access-control-template',
    '覆盖缺失onlyOwner、未初始化管理员、错误授权等权限问题。',
    'access-control',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    address public owner = address({{owner_seed}});
    function privileged() external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"越权调用后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"missing-only-owner","options":["missing-only-owner","uninitialized-admin","role-confusion"]},{"key":"owner_seed","label":"初始Owner地址","type":"text","default":"0"}]}'::jsonb,
    '[{"name":"缺失onlyOwner","params":{"variant":"missing-only-owner","owner_seed":"0"},"suggested_difficulty":1},{"name":"未初始化管理员","params":{"variant":"uninitialized-admin","owner_seed":"0"},"suggested_difficulty":2},{"name":"角色混淆","params":{"variant":"role-confusion","owner_seed":"0"},"suggested_difficulty":3}]'::jsonb,
    '[{"name":"Parity multisig library","date":"2017-07","loss":"约15万ETH"}]'::jsonb,
    '{"min":1,"max":4}'::jsonb
),
(
    1780000000520005,
    '预言机操纵模板',
    'oracle-manipulation-template',
    '覆盖现货价格、TWAP窗口、流动性不足等价格操纵路径。',
    'oracle-manipulation',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    uint256 public referencePrice = {{reference_price}};
    function manipulate() external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"价格操纵路径完成后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"spot-price","options":["spot-price","twap-bypass","thin-liquidity"]},{"key":"reference_price","label":"参考价格","type":"number","default":"100"}]}'::jsonb,
    '[{"name":"现货价格操纵","params":{"variant":"spot-price","reference_price":"100"},"suggested_difficulty":2},{"name":"TWAP绕过","params":{"variant":"twap-bypass","reference_price":"100"},"suggested_difficulty":4},{"name":"低流动性池","params":{"variant":"thin-liquidity","reference_price":"100"},"suggested_difficulty":3}]'::jsonb,
    '[{"name":"Mango Markets","date":"2022-10","loss":"约1.16亿美元"}]'::jsonb,
    '{"min":2,"max":5}'::jsonb
),
(
    1780000000520006,
    '未检查外部调用模板',
    'unchecked-call-template',
    '覆盖call返回值未检查导致的状态不一致问题。',
    'unchecked-call',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    address public target = address({{target_seed}});
    function payout() external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"未检查调用路径完成后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"low-level-call","options":["low-level-call","send-failure","delegate-return"]},{"key":"target_seed","label":"目标地址种子","type":"text","default":"0"}]}'::jsonb,
    '[{"name":"低级call未检查","params":{"variant":"low-level-call","target_seed":"0"},"suggested_difficulty":2},{"name":"send失败未处理","params":{"variant":"send-failure","target_seed":"0"},"suggested_difficulty":2}]'::jsonb,
    '[]'::jsonb,
    '{"min":2,"max":4}'::jsonb
),
(
    1780000000520007,
    'delegatecall注入模板',
    'delegatecall-injection-template',
    '覆盖delegatecall目标可控导致的上下文劫持。',
    'delegatecall',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    address public libraryTarget = address({{library_seed}});
    function execute(address) external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"delegatecall利用后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"user-supplied-library","options":["user-supplied-library","storage-overwrite","proxy-context"]},{"key":"library_seed","label":"库地址种子","type":"text","default":"0"}]}'::jsonb,
    '[{"name":"用户控制库地址","params":{"variant":"user-supplied-library","library_seed":"0"},"suggested_difficulty":3},{"name":"存储覆盖","params":{"variant":"storage-overwrite","library_seed":"0"},"suggested_difficulty":4}]'::jsonb,
    '[{"name":"Parity wallet freeze","date":"2017-11","loss":"约51万ETH冻结"}]'::jsonb,
    '{"min":3,"max":5}'::jsonb
),
(
    1780000000520008,
    '自毁攻击模板',
    'selfdestruct-template',
    '覆盖强制转账、代码销毁和生命周期误用问题。',
    'selfdestruct',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    uint256 public expectedBalance = {{expected_balance}} ether;
    function trigger() external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"自毁攻击路径完成后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"forced-ether","options":["forced-ether","code-removal","lifecycle-bypass"]},{"key":"expected_balance","label":"预期余额 ETH","type":"number","default":"1"}]}'::jsonb,
    '[{"name":"强制转账","params":{"variant":"forced-ether","expected_balance":"1"},"suggested_difficulty":2},{"name":"代码销毁","params":{"variant":"code-removal","expected_balance":"1"},"suggested_difficulty":3}]'::jsonb,
    '[]'::jsonb,
    '{"min":2,"max":4}'::jsonb
),
(
    1780000000520009,
    'tx.origin钓鱼模板',
    'tx-origin-template',
    '覆盖使用tx.origin进行鉴权导致的钓鱼攻击。',
    'tx-origin',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    address public trusted = address({{trusted_seed}});
    function claim() external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"tx.origin钓鱼路径完成后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"phishing-wallet","options":["phishing-wallet","meta-transaction","nested-call"]},{"key":"trusted_seed","label":"受信地址种子","type":"text","default":"0"}]}'::jsonb,
    '[{"name":"钱包钓鱼","params":{"variant":"phishing-wallet","trusted_seed":"0"},"suggested_difficulty":1},{"name":"嵌套调用","params":{"variant":"nested-call","trusted_seed":"0"},"suggested_difficulty":2}]'::jsonb,
    '[]'::jsonb,
    '{"min":1,"max":3}'::jsonb
),
(
    1780000000520010,
    'AMM价格操纵模板',
    'amm-price-manipulation-template',
    '覆盖恒定乘积池储备量操纵和错误报价。',
    'price-manipulation',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    uint256 public reserve = {{reserve_amount}} ether;
    function swapAndExploit() external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"AMM价格操纵完成后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"constant-product","options":["constant-product","low-liquidity","sandwich"]},{"key":"reserve_amount","label":"储备量 ETH","type":"number","default":"100"}]}'::jsonb,
    '[{"name":"恒定乘积池","params":{"variant":"constant-product","reserve_amount":"100"},"suggested_difficulty":2},{"name":"低流动性池","params":{"variant":"low-liquidity","reserve_amount":"10"},"suggested_difficulty":3},{"name":"三明治路径","params":{"variant":"sandwich","reserve_amount":"50"},"suggested_difficulty":4}]'::jsonb,
    '[]'::jsonb,
    '{"min":2,"max":5}'::jsonb
),
(
    1780000000520011,
    '未初始化代理模板',
    'uninitialized-proxy-template',
    '覆盖代理或实现合约未初始化导致的权限接管。',
    'uninitialized-proxy',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    address public admin = address({{admin_seed}});
    function initialize() external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"未初始化代理被接管后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"proxy-admin","options":["proxy-admin","implementation-takeover","initializer-replay"]},{"key":"admin_seed","label":"管理员种子","type":"text","default":"0"}]}'::jsonb,
    '[{"name":"代理管理员接管","params":{"variant":"proxy-admin","admin_seed":"0"},"suggested_difficulty":2},{"name":"实现合约接管","params":{"variant":"implementation-takeover","admin_seed":"0"},"suggested_difficulty":3}]'::jsonb,
    '[{"name":"Wormhole uninitialized proxy","date":"2022-02","loss":"白帽披露"}]'::jsonb,
    '{"min":2,"max":4}'::jsonb
),
(
    1780000000520012,
    '签名重放模板',
    'signature-replay-template',
    '覆盖缺失nonce、chainId或域分隔符导致的签名重放。',
    'signature-replay',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    uint256 public nonce = {{nonce_seed}};
    function replay(bytes calldata) external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"签名重放成功后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"missing-nonce","options":["missing-nonce","missing-chain-id","domain-confusion"]},{"key":"nonce_seed","label":"Nonce种子","type":"number","default":"0"}]}'::jsonb,
    '[{"name":"缺失nonce","params":{"variant":"missing-nonce","nonce_seed":"0"},"suggested_difficulty":2},{"name":"缺失chainId","params":{"variant":"missing-chain-id","nonce_seed":"0"},"suggested_difficulty":3},{"name":"域分隔混淆","params":{"variant":"domain-confusion","nonce_seed":"0"},"suggested_difficulty":4}]'::jsonb,
    '[]'::jsonb,
    '{"min":2,"max":5}'::jsonb
),
(
    1780000000520013,
    '抢跑交易模板',
    'front-running-template',
    '覆盖公开内存池、提交揭示缺陷和排序依赖。',
    'front-running',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    bytes32 public commitment = bytes32({{commitment_seed}});
    function reveal() external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"抢跑路径完成后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"public-solution","options":["public-solution","commit-reveal-bug","auction-ordering"]},{"key":"commitment_seed","label":"承诺种子","type":"text","default":"0"}]}'::jsonb,
    '[{"name":"公开答案抢跑","params":{"variant":"public-solution","commitment_seed":"0"},"suggested_difficulty":1},{"name":"提交揭示缺陷","params":{"variant":"commit-reveal-bug","commitment_seed":"0"},"suggested_difficulty":3}]'::jsonb,
    '[]'::jsonb,
    '{"min":1,"max":4}'::jsonb
),
(
    1780000000520014,
    '存储碰撞模板',
    'storage-collision-template',
    '覆盖代理模式下slot冲突导致的状态覆盖。',
    'storage-collision',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    bytes32 public slotKey = bytes32({{slot_seed}});
    function collide() external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"存储碰撞完成后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"proxy-slot","options":["proxy-slot","mapping-slot","diamond-storage"]},{"key":"slot_seed","label":"Slot种子","type":"text","default":"0"}]}'::jsonb,
    '[{"name":"代理slot冲突","params":{"variant":"proxy-slot","slot_seed":"0"},"suggested_difficulty":3},{"name":"mapping slot覆盖","params":{"variant":"mapping-slot","slot_seed":"0"},"suggested_difficulty":4}]'::jsonb,
    '[]'::jsonb,
    '{"min":3,"max":5}'::jsonb
),
(
    1780000000520015,
    '随机数预测模板',
    'randomness-template',
    '覆盖区块属性随机数、可预测种子和矿工可控随机。',
    'randomness',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    uint256 public seed = {{seed_value}};
    function guess(uint256) external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"随机数预测成功后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"block-timestamp","options":["block-timestamp","blockhash","predictable-seed"]},{"key":"seed_value","label":"种子值","type":"number","default":"42"}]}'::jsonb,
    '[{"name":"时间戳随机数","params":{"variant":"block-timestamp","seed_value":"42"},"suggested_difficulty":1},{"name":"区块哈希随机数","params":{"variant":"blockhash","seed_value":"42"},"suggested_difficulty":2},{"name":"可预测种子","params":{"variant":"predictable-seed","seed_value":"42"},"suggested_difficulty":2}]'::jsonb,
    '[]'::jsonb,
    '{"min":1,"max":3}'::jsonb
),
(
    1780000000520016,
    '拒绝服务模板',
    'denial-of-service-template',
    '覆盖循环Gas耗尽、拒绝接收ETH、状态阻塞等DoS问题。',
    'denial-of-service',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    uint256 public participantCount = {{participant_count}};
    function blockProgress() external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"DoS路径触发后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"gas-loop","options":["gas-loop","reject-ether","state-lock"]},{"key":"participant_count","label":"参与者数量","type":"number","default":"100"}]}'::jsonb,
    '[{"name":"循环Gas耗尽","params":{"variant":"gas-loop","participant_count":"100"},"suggested_difficulty":2},{"name":"拒绝接收ETH","params":{"variant":"reject-ether","participant_count":"10"},"suggested_difficulty":2},{"name":"状态锁死","params":{"variant":"state-lock","participant_count":"20"},"suggested_difficulty":3}]'::jsonb,
    '[{"name":"King of the Ether Throne","date":"2016-02","loss":"游戏资金锁定"}]'::jsonb,
    '{"min":2,"max":4}'::jsonb
),
(
    1780000000520017,
    '授权竞态模板',
    'approval-race-template',
    '覆盖ERC20 approve竞态和额度复用问题。',
    'approval-race',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    uint256 public allowanceValue = {{allowance_value}};
    function spend() external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"授权竞态利用后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"approve-race","options":["approve-race","permit-reuse","spender-confusion"]},{"key":"allowance_value","label":"授权额度","type":"number","default":"1000"}]}'::jsonb,
    '[{"name":"approve竞态","params":{"variant":"approve-race","allowance_value":"1000"},"suggested_difficulty":1},{"name":"permit复用","params":{"variant":"permit-reuse","allowance_value":"1000"},"suggested_difficulty":3}]'::jsonb,
    '[]'::jsonb,
    '{"min":1,"max":4}'::jsonb
),
(
    1780000000520018,
    'ERC777回调模板',
    'erc777-hook-template',
    '覆盖ERC777 tokensReceived回调引入的重入和状态同步问题。',
    'erc777-hook',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    uint256 public tokenPool = {{token_pool}};
    function tokensReceived() external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"ERC777回调利用后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"hook-reentrancy","options":["hook-reentrancy","operator-hook","exchange-hook"]},{"key":"token_pool","label":"Token池","type":"number","default":"10000"}]}'::jsonb,
    '[{"name":"回调重入","params":{"variant":"hook-reentrancy","token_pool":"10000"},"suggested_difficulty":3},{"name":"交易所回调","params":{"variant":"exchange-hook","token_pool":"50000"},"suggested_difficulty":4}]'::jsonb,
    '[{"name":"Uniswap/Lendf.Me ERC777 incident","date":"2020-04","loss":"约2500万美元"}]'::jsonb,
    '{"min":3,"max":5}'::jsonb
),
(
    1780000000520019,
    '跨链桥验证模板',
    'bridge-validation-template',
    '覆盖跨链消息验证、签名阈值和资产铸造约束问题。',
    'bridge-validation',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    uint256 public validatorThreshold = {{validator_threshold}};
    function relay(bytes calldata) external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"跨链桥验证绕过后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"signature-threshold","options":["signature-threshold","message-replay","mint-bypass"]},{"key":"validator_threshold","label":"验证阈值","type":"number","default":"2"}]}'::jsonb,
    '[{"name":"签名阈值绕过","params":{"variant":"signature-threshold","validator_threshold":"2"},"suggested_difficulty":4},{"name":"跨链消息重放","params":{"variant":"message-replay","validator_threshold":"3"},"suggested_difficulty":4},{"name":"铸造约束绕过","params":{"variant":"mint-bypass","validator_threshold":"3"},"suggested_difficulty":5}]'::jsonb,
    '[{"name":"Ronin Bridge","date":"2022-03","loss":"约6.24亿美元"}]'::jsonb,
    '{"min":4,"max":5}'::jsonb
),
(
    1780000000520020,
    '治理攻击模板',
    'governance-attack-template',
    '覆盖治理投票权快照、提案执行和闪电贷治理攻击。',
    'governance-attack',
    $sol$pragma solidity ^0.8.20;
contract Challenge {
    bool public solved;
    uint256 public quorum = {{quorum_value}};
    function executeProposal() external { solved = true; }
}$sol$,
    '[{"assertion_type":"storage_check","target":"solved","operator":"eq","expected_value":"true","description":"治理攻击执行后 solved 应为 true","extra_params":{},"sort_order":1}]'::jsonb,
    '[]'::jsonb,
    '{"params":[{"key":"variant","label":"变体","type":"select","default":"flash-governance","options":["flash-governance","snapshot-bypass","proposal-timelock"]},{"key":"quorum_value","label":"法定票数","type":"number","default":"100000"}]}'::jsonb,
    '[{"name":"闪电贷治理","params":{"variant":"flash-governance","quorum_value":"100000"},"suggested_difficulty":4},{"name":"快照绕过","params":{"variant":"snapshot-bypass","quorum_value":"50000"},"suggested_difficulty":4},{"name":"时间锁执行","params":{"variant":"proposal-timelock","quorum_value":"100000"},"suggested_difficulty":5}]'::jsonb,
    '[{"name":"Beanstalk governance exploit","date":"2022-04","loss":"约1.82亿美元"}]'::jsonb,
    '{"min":4,"max":5}'::jsonb
)
ON CONFLICT (code) DO NOTHING;
