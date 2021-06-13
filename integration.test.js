const util = require('util')
const axios = require('axios')
const exec = util.promisify(require('child_process').exec)

jest.setTimeout(60000)

const validRequestOptions = (port) => {
    return {
	method: 'post',
	url: `http://ec2-35-183-71-175.ca-central-1.compute.amazonaws.com:${port}`,
	headers: {
	    'Content-Type': 'application/json'
	},
	data: {
	    data: "some string"
	}
    }
}

const reversedString = '{"data":"gnirts emos"}'
const businessServerNotAvailable = '{"error":"business server not available"}'
const requestBadlyFormed = '{"error":"Request body contains badly-formed JSON (at position 1)"}'
const requestBodyEmpty = '{"error":"Request body must not be empty"}'
const requestContentHeaderNotJSON = '{"error":"Content-Type header is not application/json"}'
const requestBodyTooLarge = '{"error":"Can\'t read request body: http: request body too large"}'
const requestIntegerPayload = '{"error":"data is an int and not a string"}'

describe('server-status tests', () => {
    test('all servers online - expect reversed string', async () => {
    	await exec("./start_server --all")
    	const response = await axios(validRequestOptions(8000))
    	expect(JSON.stringify(response.data)).toBe(reversedString)
    })

    test('business_one offline - expect reversed string', async () => {
    	await exec("./start_server --all")
    	await exec("./stop_server business_one")
    	const response = await axios(validRequestOptions(8000))
    	expect(JSON.stringify(response.data)).toBe(reversedString)
    })

    test('business_two offline - expect reversed string', async () => {
    	await exec("./start_server --all")
    	await exec("./stop_server business_two")
    	const response = await axios(validRequestOptions(8000))
    	expect(JSON.stringify(response.data)).toBe(reversedString)
    })

    test('both servers offline - expect error message', async () => {
    	try {
    	    await exec("./start_server --all")
    	    await exec("./stop_server --business")
    	    await axios(validRequestOptions(8000))
    	} catch (error) {
    	    if(error.response) {
    		const response = error.response
    		expect(response.status).toBe(504)
    		expect(JSON.stringify(response.data)).toBe(businessServerNotAvailable)
    	    }
    	}
    })

    test('business_one online after 20 seconds - expect reversed string', async () => {
    	await exec("./start_server --all")
    	await exec("./stop_server --business")
    	setTimeout(function() {
    	    exec("./start_server business_one")
    	}, 20000)
    	const response = await axios(validRequestOptions(8000))
    	expect(JSON.stringify(response.data)).toBe(reversedString)
    })

    test('business_two online after 20 seconds - expect reversed string', async () => {
    	await exec("./start_server --all")
    	await exec("./stop_server --business")
    	setTimeout(function() {
    	    exec("./start_server business_two")
    	}, 20000)
    	const response = await axios(validRequestOptions(8000))
    	expect(JSON.stringify(response.data)).toBe(reversedString)
    })

    test('business_one online after 30 seconds - expect error message', async () => {
    	try {
    	    await exec("./start_server --all")
    	    await exec("./stop_server --business")
    	    setTimeout(function() {
    		exec("./start_server business_one")
    	    }, 30000)
    	    await axios(validRequestOptions(8000))
    	} catch (error) {
    	    if(error.response) {
    		const response = error.response
    		expect(response.status).toBe(504)
    		expect(JSON.stringify(response.data)).toBe(businessServerNotAvailable)
    	    }
    	}
    })

    test('business_two online after 30 seconds - expect error message', async () => {
    	try {
    	    await exec("./start_server --all")
    	    await exec("./stop_server --business")
    	    setTimeout(function() {
    		exec("./start_server business_two")
    	    }, 30000)
    	    await axios(validRequestOptions(8000))
    	} catch (error) {
    	    if(error.response) {
    		const response = error.response
    		expect(response.status).toBe(504)
    		expect(JSON.stringify(response.data)).toBe(businessServerNotAvailable)
    	    }
    	}
    })
})

describe('server access tests', () => {
    test('directly hit business_one server without load balancer - expect no response', async () => {
	try {
    	    await exec("./start_server --all")
    	    const response = await axios({
		...validRequestOptions(8001),
		timeout: 10000
	    })
    	} catch (error) {
	    expect(error.response).toBe(undefined)
	    expect(error.isAxiosError).toBe(true)
    	}
    })

    test('directly hit business_two server without load balancer - expect no response', async () => {
	try {
    	    await exec("./start_server --all")
    	    const response = await axios({
		...validRequestOptions(8002),
		timeout: 10000
	    })
    	} catch (error) {
	    expect(error.response).toBe(undefined)
	    expect(error.isAxiosError).toBe(true)
    	}
    })
})

describe('request formatting tests', () => {
    test('request sent with integer payload - expect error message', async () => {
    	await exec('./start_server --all')
    	const response = await axios({
    	    method: 'post',
    	    url: 'http://ec2-35-183-71-175.ca-central-1.compute.amazonaws.com:8000',
    	    headers: {
    		'Content-Type': 'application/json'
    	    },
    	    data: {
		data: 100
	    }
    	})
    	expect(JSON.stringify(response.data)).toBe(requestIntegerPayload)
    })

    test('request sent with invalid JSON - expect error message', async () => {
    	await exec('./start_server --all')
    	const response = await axios({
    	    method: 'post',
    	    url: 'http://ec2-35-183-71-175.ca-central-1.compute.amazonaws.com:8000',
    	    headers: {
    		'Content-Type': 'application/json'
    	    },
    	    data: "some string"
    	})
    	expect(JSON.stringify(response.data)).toBe(requestBadlyFormed)
    })

    test('request sent with empty body - expect error message', async () => {
    	await exec('./start_server --all')
    	const response = await axios({
    	    method: 'post',
    	    url: 'http://ec2-35-183-71-175.ca-central-1.compute.amazonaws.com:8000',
    	    headers: {
    		'Content-Type': 'application/json'
    	    }
    	})
    	expect(JSON.stringify(response.data)).toBe(requestBodyEmpty)
    })

    test('request sent with invalid headers - expect error message', async () => {
    	await exec('./start_server --all')
    	const response = await axios({
    	    method: 'post',
    	    url: 'http://ec2-35-183-71-175.ca-central-1.compute.amazonaws.com:8000',
    	    headers: {
    		'Content-Type': 'text/plain'
    	    },
    	    data: {
    		data: "some string"
    	    }
    	})
    	expect(JSON.stringify(response.data)).toBe(requestContentHeaderNotJSON)
    })

    test('request sent with string too large (over 1MB) - expect error message', async () => {
    	await exec('./start_server --all')

    	let longWord = new Array(1250000)
    	longWord = longWord.fill('A')
	
    	const response = await axios({
    	    method: 'post',
    	    url: 'http://ec2-35-183-71-175.ca-central-1.compute.amazonaws.com:8000',
    	    headers: {
    		'Content-Type': 'application/json'
    	    },
    	    data: {
    		data: longWord
    	    }
    	})
	
    	expect(JSON.stringify(response.data)).toBe(requestBodyTooLarge)
    })
})
