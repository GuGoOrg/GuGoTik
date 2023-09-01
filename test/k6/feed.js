import { sleep } from 'k6'
import http from 'k6/http'

export const options = {
    scenarios: {
        Scenario_1: {
            executor: 'ramping-vus',
            gracefulStop: '30s',
            stages: [
                { target: 1000, duration: '15s' },
                { target: 1500, duration: '30s' },
                { target: 1000, duration: '15s' },
            ],
            gracefulRampDown: '30s',
            exec: 'login',
        },
    },
}

export function login() {
    http.get('https://gugotik.endymx.qzwxsaedc.cn/douyin/feed?')

    sleep(3)
}
