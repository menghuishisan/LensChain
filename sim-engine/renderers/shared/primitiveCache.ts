import type { Primitive } from "./types.js";

/**
 * PrimitiveDiffResult 描述两帧之间原语的增删改。
 */
export interface PrimitiveDiffResult {
  added: Primitive[];
  updated: Primitive[];
  removed: Primitive[];
  unchanged: Primitive[];
}

/**
 * PrimitiveCache 按 ID 跟踪原语变化，为增量渲染提供 diff 能力。
 */
export class PrimitiveCache {
  private cache = new Map<string, Primitive>();

  /**
   * applyDiff 对比新一帧原语与缓存，返回差异并更新缓存。
   */
  public applyDiff(newPrimitives: Primitive[]): PrimitiveDiffResult {
    const newIds = new Set<string>();
    const added: Primitive[] = [];
    const updated: Primitive[] = [];
    const unchanged: Primitive[] = [];
    const removed: Primitive[] = [];

    for (const p of newPrimitives) {
      newIds.add(p.id);
      const old = this.cache.get(p.id);
      if (!old) {
        added.push(p);
      } else if (!shallowEqualPrimitive(old, p)) {
        updated.push(p);
      } else {
        unchanged.push(p);
      }
      this.cache.set(p.id, p);
    }

    for (const [id, p] of this.cache) {
      if (!newIds.has(id)) {
        removed.push(p);
        this.cache.delete(id);
      }
    }

    return { added, updated, removed, unchanged };
  }

  /**
   * get 返回缓存中指定 ID 的原语。
   */
  public get(id: string): Primitive | undefined {
    return this.cache.get(id);
  }

  /**
   * list 返回当前缓存的全部原语。
   */
  public list(): Primitive[] {
    return Array.from(this.cache.values());
  }

  /**
   * clear 清空缓存。
   */
  public clear(): void {
    this.cache.clear();
  }

  /**
   * size 返回当前缓存中的原语数量。
   */
  public get size(): number {
    return this.cache.size;
  }
}

/**
 * shallowEqualPrimitive 浅对比两个原语是否相同（type + layer + params JSON）。
 */
function shallowEqualPrimitive(a: Primitive, b: Primitive): boolean {
  if (a.type !== b.type || a.layer !== b.layer) return false;
  if (a.clickable !== b.clickable) return false;
  if (a.hover_tooltip !== b.hover_tooltip) return false;
  if (a.click_action !== b.click_action) return false;
  return JSON.stringify(a.params) === JSON.stringify(b.params);
}
