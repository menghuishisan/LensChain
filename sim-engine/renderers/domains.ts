import { PrimitiveBasedRenderer } from "./shared/primitiveRenderer.js";
import type { RenderConfig, SceneCategory } from "./shared/types.js";

// ============================================================
// 8 领域渲染器 + 1 通用渲染器
// 新协议下渲染完全由 primitives 驱动，领域渲染器仅提供主题配色。
// ============================================================

export class ConsensusRenderer extends PrimitiveBasedRenderer {
  public readonly domain: SceneCategory = "consensus";
  public getTheme(): RenderConfig {
    return {
      title: "共识过程",
      theme: {
        background: "#0d1222",
        foreground: "#f4f7fb",
        accent: "#63d2ff",
        success: "#47c98b",
        warning: "#ffc857",
        danger: "#ff6b6b",
        muted: "#93a4bb",
        grid: "rgba(147, 164, 187, 0.16)"
      }
    };
  }
}

export class CryptographyRenderer extends PrimitiveBasedRenderer {
  public readonly domain: SceneCategory = "cryptography";
  public getTheme(): RenderConfig {
    return {
      title: "密码学",
      theme: {
        background: "#0a1628",
        foreground: "#f4f7fb",
        accent: "#a78bfa",
        success: "#47c98b",
        warning: "#ffc857",
        danger: "#ff6b6b",
        muted: "#93a4bb",
        grid: "rgba(147, 164, 187, 0.14)"
      }
    };
  }
}

export class NodeNetworkRenderer extends PrimitiveBasedRenderer {
  public readonly domain: SceneCategory = "node_network";
  public getTheme(): RenderConfig {
    return {
      title: "节点与网络",
      theme: {
        background: "#0b1527",
        foreground: "#f4f7fb",
        accent: "#38bdf8",
        success: "#47c98b",
        warning: "#ffc857",
        danger: "#ff6b6b",
        muted: "#93a4bb",
        grid: "rgba(147, 164, 187, 0.14)"
      }
    };
  }
}

export class DataStructureRenderer extends PrimitiveBasedRenderer {
  public readonly domain: SceneCategory = "data_structure";
  public getTheme(): RenderConfig {
    return {
      title: "数据结构",
      theme: {
        background: "#0c1425",
        foreground: "#f4f7fb",
        accent: "#34d399",
        success: "#47c98b",
        warning: "#ffc857",
        danger: "#ff6b6b",
        muted: "#93a4bb",
        grid: "rgba(147, 164, 187, 0.14)"
      }
    };
  }
}

export class TransactionRenderer extends PrimitiveBasedRenderer {
  public readonly domain: SceneCategory = "transaction";
  public getTheme(): RenderConfig {
    return {
      title: "交易",
      theme: {
        background: "#0d1220",
        foreground: "#f4f7fb",
        accent: "#fb923c",
        success: "#47c98b",
        warning: "#ffc857",
        danger: "#ff6b6b",
        muted: "#93a4bb",
        grid: "rgba(147, 164, 187, 0.14)"
      }
    };
  }
}

export class SmartContractRenderer extends PrimitiveBasedRenderer {
  public readonly domain: SceneCategory = "smart_contract";
  public getTheme(): RenderConfig {
    return {
      title: "智能合约",
      theme: {
        background: "#0e1424",
        foreground: "#f4f7fb",
        accent: "#f472b6",
        success: "#47c98b",
        warning: "#ffc857",
        danger: "#ff6b6b",
        muted: "#93a4bb",
        grid: "rgba(147, 164, 187, 0.14)"
      }
    };
  }
}

export class AttackSecurityRenderer extends PrimitiveBasedRenderer {
  public readonly domain: SceneCategory = "attack_security";
  public getTheme(): RenderConfig {
    return {
      title: "攻防安全",
      theme: {
        background: "#120e1c",
        foreground: "#f4f7fb",
        accent: "#ef4444",
        success: "#47c98b",
        warning: "#ffc857",
        danger: "#ff6b6b",
        muted: "#93a4bb",
        grid: "rgba(147, 164, 187, 0.14)"
      }
    };
  }
}

export class EconomicRenderer extends PrimitiveBasedRenderer {
  public readonly domain: SceneCategory = "economic";
  public getTheme(): RenderConfig {
    return {
      title: "经济模型",
      theme: {
        background: "#0d1422",
        foreground: "#f4f7fb",
        accent: "#facc15",
        success: "#47c98b",
        warning: "#ffc857",
        danger: "#ff6b6b",
        muted: "#93a4bb",
        grid: "rgba(147, 164, 187, 0.14)"
      }
    };
  }
}

/**
 * GenericPrimitiveRenderer 通用兜底渲染器。
 * 用于未匹配到具体领域的场景，使用默认主题。
 */
export class GenericPrimitiveRenderer extends PrimitiveBasedRenderer {
  public readonly domain: SceneCategory = "generic";
  public getTheme(): RenderConfig {
    return {
      title: "通用场景",
      theme: {
        background: "#09111f",
        foreground: "#f4f7fb",
        accent: "#63d2ff",
        success: "#47c98b",
        warning: "#ffc857",
        danger: "#ff6b6b",
        muted: "#93a4bb",
        grid: "rgba(147, 164, 187, 0.12)"
      }
    };
  }
}
