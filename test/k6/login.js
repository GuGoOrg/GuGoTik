import { sleep } from 'k6'
import http from 'k6/http'

export const options = {
    scenarios: {
        Scenario_1: {
            executor: 'ramping-vus',
            gracefulStop: '30s',
            stages: [
                { target: 100, duration: '15s' },
                { target: 200, duration: '30s' },
                { target: 100, duration: '15s' },
            ],
            gracefulRampDown: '30s',
            exec: 'login',
        },
    },
}

export function login() {
    http.post('https://gugotik.endymx.qzwxsaedc.cn/douyin/user/login?username=epicmo&password=epicmo')

    sleep(3)
}
