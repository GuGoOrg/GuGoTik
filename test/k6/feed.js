import { sleep } from 'k6'
import http from 'k6/http'

export const options = {
    scenarios: {
        Scenario_1: {
            executor: 'ramping-vus',
            gracefulStop: '30s',
            stages: [
                { target: 1000, duration: '60s' },
            ],
            gracefulRampDown: '30s',
            exec: 'feed',
        },
    },
}

export function feed() {
    http.get('https://gugotik.endymx.qzwxsaedc.cn/douyin/feed?')

    sleep(3)
}
