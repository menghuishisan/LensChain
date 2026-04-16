import { BaseCanvasRenderer } from "../shared/baseCanvasRenderer.js";
import { asArray, asNumber, asObject, asString } from "../shared/utils.js";
import type { RenderConfig, RenderContext, RenderState } from "../shared/types.js";

/**
 * DATA_STRUCTURE_ALGORITHMS 定义数据结构领域支持的算法类型。
 */
export const DATA_STRUCTURE_ALGORITHMS = [
  "blockchain-structure",
  "block-internal",
  "mpt-trie",
  "bloom-filter",
  "dht-storage"
] as const;

/**
 * DataStructureRenderer 实现数据结构领域渲染器。
 */
export class DataStructureRenderer extends BaseCanvasRenderer {
  public readonly domain = "data_structure" as const;
  public readonly supportedAlgorithms = DATA_STRUCTURE_ALGORITHMS;

  /**
   * getRenderConfig 返回数据结构领域的渲染配置。
   */
  public getRenderConfig(algorithmType: string): RenderConfig {
    return {
      algorithmType,
      title: "数据结构",
      subtitle: "链式结构、树形结构与区块内部展开",
      theme: {
        background: "#0d1514",
        foreground: "#ebfff9",
        accent: "#47d8b3",
        success: "#6fdb8d",
        warning: "#ffc857",
        danger: "#ff7b7b",
        muted: "#9ac3ba",
        grid: "rgba(154, 195, 186, 0.16)"
      }
    };
  }

  /**
   * render 绘制链式或树形数据结构。
   */
  public render(state: RenderState, context: RenderContext): void {
    const ctx = this.beginFrame(context, state.title, "链式结构、树形结构与哈希指针");
    this.clearOverlaySVG(context);
    const metrics = state.metrics ?? [];
    if (metrics.length > 0) {
      this.drawMetricCards(ctx, metrics.slice(0, 4));
    }
    const blocks = asArray<unknown>(asObject(state.renderData).blocks).map((item, index) =>
      asString(asObject(item).label, `B${index}`)
    );
    switch (state.algorithmType) {
      case "blockchain-structure":
        this.renderBlockchain(ctx, state, blocks, context);
        break;
      case "block-internal":
        this.renderBlockInternal(ctx, state);
        break;
      case "mpt-trie":
        this.renderTrie(ctx, state, context);
        break;
      case "bloom-filter":
        this.renderBloomFilter(ctx, state);
        break;
      case "dht-storage":
        this.renderDHT(ctx, state, context);
        break;
      default:
        this.renderBlockChainLine(ctx, blocks);
        break;
    }
    this.drawListPanel(
      ctx,
      "结构摘要",
      [
        { label: "算法", value: state.algorithmType },
        { label: "Tick", value: String(state.tick) },
        { label: "结构项", value: String(blocks.length) },
        { label: "变化项", value: String(state.changedKeys?.length ?? 0) }
      ],
      context.width - 240,
      72,
      220,
      122
    );
    this.drawChangedKeys(ctx, state.changedKeys ?? [], context.width);
    this.drawAnnotations(ctx, state);
    this.drawTooltip(ctx, state.tooltip ?? [], 24, context.height - 150);
    if ((state.timeline?.length ?? 0) > 0) {
      this.drawTimeline(ctx, context.width, context.height, state.timeline ?? []);
    }
  }

  /**
   * renderBlockChainLine 绘制通用链式结构。
   */
  private renderBlockChainLine(ctx: CanvasRenderingContext2D, blocks: string[]): void {
    blocks.forEach((block, index) => {
      const x = 48 + index * 120;
      ctx.fillStyle = "rgba(18, 31, 29, 0.94)";
      ctx.strokeStyle = "#47d8b3";
      ctx.beginPath();
      ctx.roundRect(x, 220, 92, 62, 16);
      ctx.fill();
      ctx.stroke();
      ctx.fillStyle = "#ebfff9";
      ctx.font = "600 13px sans-serif";
      ctx.fillText(block, x + 32, 255);
      if (index < blocks.length - 1) {
        ctx.strokeStyle = "#47d8b3";
        ctx.beginPath();
        ctx.moveTo(x + 92, 252);
        ctx.lineTo(x + 118, 252);
        ctx.stroke();
      }
    });
  }

  /**
   * renderBlockchain 绘制链结构和分叉最长链标记。
   */
  private renderBlockchain(
    ctx: CanvasRenderingContext2D,
    state: RenderState,
    blocks: string[],
    context: RenderContext
  ): void {
    this.renderBlockChainLine(ctx, blocks.length > 0 ? blocks : ["B0", "B1", "B2"]);
    const forks = asArray<unknown>(asObject(state.renderData).forks);
    forks.slice(0, 3).forEach((item, index) => {
      const fork = asObject(item);
      const x = asNumber(fork.x, 180 + index * 120);
      const y = asNumber(fork.y, 320);
      ctx.fillStyle = "#ffc857";
      ctx.beginPath();
      ctx.roundRect(x, y, 88, 34, 12);
      ctx.fill();
      ctx.fillStyle = "#0d1514";
      ctx.font = "11px sans-serif";
      ctx.fillText(asString(fork.label, `分叉${index + 1}`), x + 14, y + 21);
    });
    this.renderOverlaySVG(context, `<path d="M140 252 C180 252 162 338 220 338" fill="none" stroke="rgba(255,200,87,0.6)" stroke-width="3" />`);
  }

  /**
   * renderBlockInternal 绘制区块头、交易体和 Merkle 根展开。
   */
  private renderBlockInternal(ctx: CanvasRenderingContext2D, state: RenderState): void {
    const fields = ["parent_hash", "merkle_root", "timestamp", "nonce", "difficulty"];
    fields.forEach((field, index) => {
      const y = 178 + index * 34;
      ctx.fillStyle = "rgba(18, 31, 29, 0.94)";
      ctx.strokeStyle = field === "merkle_root" ? "#ffc857" : "#47d8b3";
      ctx.beginPath();
      ctx.roundRect(52, y, 260, 26, 10);
      ctx.fill();
      ctx.stroke();
      ctx.fillStyle = "#ebfff9";
      ctx.font = "11px sans-serif";
      ctx.fillText(`${field}: ${asString(asObject(state.renderData)[field], "0x...")}`, 66, y + 17);
    });
    this.drawProcessStages(ctx, ["字段展开", "Merkle 根", "Nonce 关系"], Math.min(2, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 356, 224, 240);
  }

  /**
   * renderTrie 绘制 MPT 路径查找与状态更新传播。
   */
  private renderTrie(ctx: CanvasRenderingContext2D, state: RenderState, context: RenderContext): void {
    const nodes = asArray<unknown>(asObject(state.renderData).trie_nodes);
    nodes.slice(0, 9).forEach((item, index) => {
      const node = asObject(item);
      const x = 82 + (index % 3) * 120;
      const y = 178 + Math.floor(index / 3) * 72;
      ctx.fillStyle = asString(node.status, "") === "changed" ? "#ffc857" : "#47d8b3";
      ctx.beginPath();
      ctx.roundRect(x, y, 76, 30, 12);
      ctx.fill();
      ctx.fillStyle = "#0d1514";
      ctx.font = "11px sans-serif";
      ctx.fillText(asString(node.label, `node${index}`), x + 12, y + 19);
    });
    this.renderOverlaySVG(context, `<path d="M120 208 L240 280 L360 208" fill="none" stroke="rgba(71,216,179,0.5)" stroke-width="3" />`);
    this.drawProcessStages(ctx, ["路径查找", "节点展开", "哈希回写"], Math.min(2, asNumber(asObject(state.renderData).phase_index, 1)), asNumber(asObject(state.renderData).progress, 0.5), 60, 412, 300);
  }

  /**
   * renderBloomFilter 绘制位数组、多哈希映射和误判率。
   */
  private renderBloomFilter(ctx: CanvasRenderingContext2D, state: RenderState): void {
    const bits = asArray<unknown>(asObject(state.renderData).bits);
    for (let index = 0; index < 32; index += 1) {
      const x = 48 + (index % 16) * 28;
      const y = 198 + Math.floor(index / 16) * 34;
      const active = Boolean(bits[index]);
      ctx.fillStyle = active ? "#47d8b3" : "rgba(154, 195, 186, 0.28)";
      ctx.fillRect(x, y, 20, 20);
    }
    this.drawComparisonPanel(ctx, 48, 300, 420, 94, "哈希函数", "查询结果", [
      { label: "h1", value: String(asNumber(asObject(state.renderData).h1, 3)) },
      { label: "h2", value: String(asNumber(asObject(state.renderData).h2, 9)) }
    ], [
      { label: "可能存在", value: asString(asObject(state.renderData).result, "是") },
      { label: "误判率", value: asString(asObject(state.renderData).false_positive_rate, "1.2%") }
    ]);
  }

  /**
   * renderDHT 绘制 DHT 环形空间与键值映射。
   */
  private renderDHT(ctx: CanvasRenderingContext2D, state: RenderState, context: RenderContext): void {
    const centerX = 230;
    const centerY = 270;
    const radius = 100;
    ctx.strokeStyle = "#47d8b3";
    ctx.beginPath();
    ctx.arc(centerX, centerY, radius, 0, Math.PI * 2);
    ctx.stroke();
    const nodes = asArray<unknown>(asObject(state.renderData).dht_nodes);
    nodes.slice(0, 12).forEach((item, index) => {
      const angle = (index / Math.max(1, nodes.length)) * Math.PI * 2;
      const x = centerX + Math.cos(angle) * radius;
      const y = centerY + Math.sin(angle) * radius;
      ctx.fillStyle = "#47d8b3";
      ctx.beginPath();
      ctx.arc(x, y, 8, 0, Math.PI * 2);
      ctx.fill();
      ctx.fillStyle = "#ebfff9";
      ctx.font = "10px sans-serif";
      ctx.fillText(asString(asObject(item).label, `N${index}`), x + 10, y);
    });
    this.renderOverlaySVG(context, `<path d="M230 270 L310 210" stroke="rgba(255,200,87,0.7)" stroke-width="4" />`);
  }
}
