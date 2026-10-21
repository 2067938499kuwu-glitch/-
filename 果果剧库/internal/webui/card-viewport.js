import { reconcileChildren, withFocus } from './ui-core.js';

export function createCardViewport({container, scroller, create, update, key, rendered}) {
  let items = [], emptyNode = null, frame = 0, revision = 0, drawnRevision = -1;
  let mounted = new Map();
  container.setAttribute('role', 'list');

  function schedule() {
    if (!frame) frame = requestAnimationFrame(() => {frame = 0; draw();});
  }

  function draw() {
    if (drawnRevision === revision) return;
    const nextMounted = new Map(), nodes = [], changed = [];
    for (let index = 0; index < items.length; index++) {
      const item = items[index], renderKey = key(item);
      let card = mounted.get(item.id);
      if (!card || card.renderKey !== renderKey) {
        card = create(item);
      } else {
        update(card, item);
      }
      changed.push(card);
      card.setAttribute('role', 'listitem');
      card.setAttribute('aria-posinset', String(index + 1));
      card.setAttribute('aria-setsize', String(items.length));
      nextMounted.set(item.id, card);
      nodes.push(card);
    }
    if (!items.length && emptyNode) nodes.push(emptyNode);
    const active = document.activeElement?.closest?.('.card');
    if (active && container.contains(active) && !nextMounted.has(active.dataset.dramaId)) scroller.focus({preventScroll: true});
    withFocus(container, () => reconcileChildren(container, nodes));
    mounted = nextMounted;
    drawnRevision = revision;
    container.setAttribute('aria-busy', 'false');
    if (changed.length) rendered(changed);
  }

  return {
    setItems(next, placeholder = null) {
      items = next;
      emptyNode = placeholder;
      revision++;
      container.setAttribute('aria-busy', 'true');
      schedule();
    },
    refresh() {
      drawnRevision = -1;
      schedule();
    }
  };
}
