package querybuilder_test

import (
	"fmt"

	"fli/internal/querybuilder"
)

func Example_count() {
	schema := &querybuilder.VPCFlowLogsSchema{}
	q, _ := querybuilder.New(schema,
		querybuilder.WithVerb(querybuilder.VerbCount),
		querybuilder.WithFields("srcaddr"),
		querybuilder.WithGroupBy("srcaddr"),
		querybuilder.WithFilter(&querybuilder.Eq{Field: "interface_id", Value: "eni-123"}),
		querybuilder.WithLimit(10),
	)
	fmt.Println(q.String())
	// Output:
	// parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status | filter interface_id = 'eni-123' | stats count(srcaddr) as srcaddr_count by srcaddr | sort srcaddr_count desc | limit 10
}

func Example_sum() {
	schema := &querybuilder.VPCFlowLogsSchema{}
	q, _ := querybuilder.New(schema,
		querybuilder.WithVerb(querybuilder.VerbSum),
		querybuilder.WithFields("bytes"),
		querybuilder.WithGroupBy("srcaddr"),
		querybuilder.WithFilter(&querybuilder.Eq{Field: "action", Value: "ACCEPT"}),
		querybuilder.WithLimit(10),
	)
	fmt.Println(q.String())
	// Output:
	// parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status | filter action = 'ACCEPT' | stats sum(bytes) as bytes_sum by srcaddr | sort bytes_sum desc | limit 10
}

func Example_v3() {
	schema := &querybuilder.VPCFlowLogsSchema{}
	q, _ := querybuilder.New(schema,
		querybuilder.WithVersion(3),
		querybuilder.WithVerb(querybuilder.VerbSum),
		querybuilder.WithFields("bytes"),
		querybuilder.WithGroupBy("vpc_id", "subnet_id"),
		querybuilder.WithFilter(&querybuilder.Eq{Field: "srcaddr", Value: "10.0.0.1"}),
		querybuilder.WithLimit(5),
	)
	fmt.Println(q.String())
	// Output:
	// parse @message "* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status, vpc_id, subnet_id, instance_id, tcp_flags, type, pkt_srcaddr, pkt_dstaddr, region, az_id, sublocation_type, sublocation_id, pkt_src_aws_service, pkt_dst_aws_service, flow_direction, traffic_path | filter srcaddr = '10.0.0.1' | stats sum(bytes) as bytes_sum by vpc_id, subnet_id | sort bytes_sum desc | limit 5
}

func Example_v5() {
	schema := &querybuilder.VPCFlowLogsSchema{}
	q, _ := querybuilder.New(schema,
		querybuilder.WithVersion(5),
		querybuilder.WithVerb(querybuilder.VerbSum),
		querybuilder.WithFields("bytes"),
		querybuilder.WithGroupBy("flow_direction"),
		querybuilder.WithFilter(&querybuilder.Eq{Field: "pkt_src_aws_service", Value: "EC2"}),
		querybuilder.WithLimit(5),
	)
	fmt.Println(q.String())
	// Output:
	// parse @message "* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status, vpc_id, subnet_id, instance_id, tcp_flags, type, pkt_srcaddr, pkt_dstaddr, region, az_id, sublocation_type, sublocation_id, pkt_src_aws_service, pkt_dst_aws_service, flow_direction, traffic_path | filter pkt_src_aws_service = 'EC2' | stats sum(bytes) as bytes_sum by flow_direction | sort bytes_sum desc | limit 5
}

func Example_basic() {
	opts := []querybuilder.Option{
		querybuilder.WithLimit(5),
		querybuilder.WithFields("srcaddr", "dstaddr", "action"),
		querybuilder.WithVerb(querybuilder.VerbRaw),
	}
	schema := &querybuilder.VPCFlowLogsSchema{}
	b, err := querybuilder.New(schema, opts...)
	if err != nil {
		fmt.Printf("failed to build: %v", err)
		return
	}
	fmt.Println(b.String())
	// Output:
	// parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status | display srcaddr, dstaddr, action | limit 5
}

func Example_complex() {
	filter, err := querybuilder.ParseFilter("(action = 'ACCEPT' or action = 'REJECT') and srcaddr like '10'")
	if err != nil {
		fmt.Printf("failed to parse filter: %v", err)
		return
	}
	opts := []querybuilder.Option{
		querybuilder.WithFilter(filter),
		querybuilder.WithFields("bytes"),
		querybuilder.WithGroupBy("dstport"),
		querybuilder.WithVerb(querybuilder.VerbSum),
		querybuilder.WithLimit(5),
	}
	schema := &querybuilder.VPCFlowLogsSchema{}
	b, err := querybuilder.New(schema, opts...)
	if err != nil {
		fmt.Printf("failed to build: %v", err)
		return
	}
	fmt.Println(b.String())
	// Output:
	// parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status | filter (action = 'ACCEPT' or action = 'REJECT') and srcaddr like '10' | stats sum(bytes) as bytes_sum by dstport | sort bytes_sum desc | limit 5
}

// Multi-field aggregation examples
func Example_multiFieldCount() {
	schema := &querybuilder.VPCFlowLogsSchema{}
	q, _ := querybuilder.New(schema,
		querybuilder.WithAggregations(
			querybuilder.AggregationField{Field: "srcaddr", Verb: querybuilder.VerbCount},
			querybuilder.AggregationField{Field: "dstaddr", Verb: querybuilder.VerbCount},
		),
		querybuilder.WithGroupBy("action"),
		querybuilder.WithLimit(10),
	)
	fmt.Println(q.String())
	// Output:
	// parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status | stats count(srcaddr) as srcaddr_count, count(dstaddr) as dstaddr_count by action | sort srcaddr_count desc | limit 10
}

func Example_multiFieldMixed() {
	schema := &querybuilder.VPCFlowLogsSchema{}
	q, _ := querybuilder.New(schema,
		querybuilder.WithAggregations(
			querybuilder.AggregationField{Field: "srcaddr", Verb: querybuilder.VerbCount},
			querybuilder.AggregationField{Field: "bytes", Verb: querybuilder.VerbSum},
			querybuilder.AggregationField{Field: "packets", Verb: querybuilder.VerbAvg},
		),
		querybuilder.WithFilter(&querybuilder.Eq{Field: "action", Value: "ACCEPT"}),
		querybuilder.WithGroupBy("protocol"),
		querybuilder.WithLimit(5),
	)
	fmt.Println(q.String())
	// Output:
	// parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status | filter action = 'ACCEPT' | stats count(srcaddr) as srcaddr_count, sum(bytes) as bytes_sum, avg(packets) as packets_avg by protocol | sort srcaddr_count desc | limit 5
}

func Example_multiFieldComputed() {
	schema := &querybuilder.VPCFlowLogsSchema{}
	q, _ := querybuilder.New(schema,
		querybuilder.WithAggregations(
			querybuilder.AggregationField{Field: "duration", Verb: querybuilder.VerbSum},
			querybuilder.AggregationField{Field: "bytes", Verb: querybuilder.VerbAvg},
		),
		querybuilder.WithGroupBy("srcaddr", "dstaddr"),
		querybuilder.WithLimit(10),
	)
	fmt.Println(q.String())
	// Output:
	// parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status | stats sum(end - start) as duration_sum, avg(bytes) as bytes_avg by srcaddr, dstaddr | sort duration_sum desc | limit 10
}

func Example_multiFieldV3() {
	schema := &querybuilder.VPCFlowLogsSchema{}
	q, _ := querybuilder.New(schema,
		querybuilder.WithVersion(3),
		querybuilder.WithAggregations(
			querybuilder.AggregationField{Field: "vpc_id", Verb: querybuilder.VerbCount},
			querybuilder.AggregationField{Field: "bytes", Verb: querybuilder.VerbSum},
			querybuilder.AggregationField{Field: "packets", Verb: querybuilder.VerbSum},
		),
		querybuilder.WithGroupBy("vpc_id", "subnet_id"),
		querybuilder.WithFilter(&querybuilder.Eq{Field: "action", Value: "ACCEPT"}),
		querybuilder.WithLimit(5),
	)
	fmt.Println(q.String())
	// Output:
	// parse @message "* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status, vpc_id, subnet_id, instance_id, tcp_flags, type, pkt_srcaddr, pkt_dstaddr, region, az_id, sublocation_type, sublocation_id, pkt_src_aws_service, pkt_dst_aws_service, flow_direction, traffic_path | filter action = 'ACCEPT' | stats count(vpc_id) as vpc_id_count, sum(bytes) as bytes_sum, sum(packets) as packets_sum by vpc_id, subnet_id | sort vpc_id_count desc | limit 5
}

func Example_multiFieldWithStar() {
	schema := &querybuilder.VPCFlowLogsSchema{}
	q, _ := querybuilder.New(schema,
		querybuilder.WithAggregations(
			querybuilder.AggregationField{Field: "*", Verb: querybuilder.VerbCount},
			querybuilder.AggregationField{Field: "bytes", Verb: querybuilder.VerbSum},
		),
		querybuilder.WithGroupBy("srcaddr"),
		querybuilder.WithLimit(10),
	)
	fmt.Println(q.String())
	// Output:
	// parse @message "* * * * * * * * * * * * * *" as version, account_id, interface_id, srcaddr, dstaddr, srcport, dstport, protocol, packets, bytes, start, end, action, log_status | stats count(*) as flows, sum(bytes) as bytes_sum by srcaddr | sort flows desc | limit 10
}
