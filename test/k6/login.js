import { sleep } from 'k6'
import http from 'k6/http'

export const options = {
    ext: {
        loadimpact: {
            distribution: { 'amazon:us:ashburn': { loadZone: 'amazon:us:ashburn', percent: 100 } },
            apm: [],
        },
    },
    thresholds: {},
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
            exec: 'scenario_1',
        },
    },
}

export function scenario_1() {
    // LoginTest
    http.post('http://127.0.0.1:37000/douyin/user/login?username=epicmo&password=epicmo')

    // Automatically added sleep
    sleep(1)
}
