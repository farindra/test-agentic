import { describe, it, expect, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { useOnline } from './useOnline'

const TestComponent = defineComponent({
  setup() {
    const { isOnline } = useOnline()
    return { isOnline }
  },
  render() {
    return h('div', String(this.isOnline))
  },
})

describe('useOnline', () => {
  let wrapper

  afterEach(() => {
    wrapper?.unmount()
  })

  it('reflects navigator.onLine on mount', () => {
    Object.defineProperty(navigator, 'onLine', { value: true, configurable: true })
    wrapper = mount(TestComponent)
    expect(wrapper.text()).toBe('true')
  })

  it('updates to false when the browser fires "offline"', async () => {
    wrapper = mount(TestComponent)
    window.dispatchEvent(new Event('offline'))
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toBe('false')
  })

  it('updates back to true when the browser fires "online"', async () => {
    wrapper = mount(TestComponent)
    window.dispatchEvent(new Event('offline'))
    await wrapper.vm.$nextTick()
    window.dispatchEvent(new Event('online'))
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toBe('true')
  })
})
